package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/amirhasanpour/task-manager/api-gateway/internal/client"
	"github.com/amirhasanpour/task-manager/api-gateway/proto"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type AuthMiddleware struct {
	userClient client.UserClient
	jwtSecret  string
	logger     *zap.Logger
}

func NewAuthMiddleware(userClient client.UserClient, jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		userClient: userClient,
		jwtSecret:  jwtSecret,
		logger:     zap.L().Named("auth_middleware"),
	}
}

func (m *AuthMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for public endpoints
		if m.isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.logger.Debug("Missing authorization header", zap.String("path", c.Request.URL.Path))
			c.AbortWithStatusJSON(401, gin.H{"error": "Authorization header is required"})
			return
		}

		// Check if it's a Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			m.logger.Debug("Invalid authorization header format", zap.String("path", c.Request.URL.Path))
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			m.logger.Debug("Empty bearer token", zap.String("path", c.Request.URL.Path))
			c.AbortWithStatusJSON(401, gin.H{"error": "Bearer token is empty"})
			return
		}

		// Validate token with user service
		user, err := m.validateToken(c.Request.Context(), tokenString)
		if err != nil {
			m.logger.Debug("Token validation failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Set user information in context
		c.Set("user_id", user.Id)
		c.Set("user_email", user.Email)
		c.Set("user_username", user.Username)
		c.Set("user_full_name", user.FullName)
		c.Set("token", tokenString)

		m.logger.Debug("User authenticated successfully",
			zap.String("user_id", user.Id),
			zap.String("path", c.Request.URL.Path),
		)

		c.Next()
	}
}

func (m *AuthMiddleware) validateToken(ctx context.Context, tokenString string) (*proto.User, error) {
	// First, try to parse and validate JWT locally for performance
	claims, err := m.parseJWT(tokenString)
	if err != nil {
		m.logger.Debug("Failed to parse JWT locally", zap.Error(err))
		// Fall back to user service validation
		return m.validateWithUserService(ctx, tokenString)
	}

	// Check if token is expired
	if !claims.Valid {
		m.logger.Debug("JWT token expired locally")
		return m.validateWithUserService(ctx, tokenString)
	}

	// Extract user info from claims
	userID, ok := claims.Claims.(jwt.MapClaims)["user_id"].(string)
	if !ok {
		m.logger.Debug("Missing user_id in JWT claims")
		return m.validateWithUserService(ctx, tokenString)
	}

	// Return user info from claims (simplified)
	return &proto.User{
		Id:       userID,
		Username: claims.Claims.(jwt.MapClaims)["username"].(string),
		Email:    claims.Claims.(jwt.MapClaims)["email"].(string),
		FullName: claims.Claims.(jwt.MapClaims)["full_name"].(string),
	}, nil
}

func (m *AuthMiddleware) parseJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return token, nil
}

func (m *AuthMiddleware) validateWithUserService(ctx context.Context, tokenString string) (*proto.User, error) {
	req := &proto.ValidateTokenRequest{
		Token: tokenString,
	}

	resp, err := m.userClient.ValidateToken(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Valid {
		return nil, errors.New("invalid token")
	}

	return resp.User, nil
}

func (m *AuthMiddleware) isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/health",
		"/metrics",
		"/swagger",
		"/api/v1/auth/register",
		"/api/v1/auth/login",
		"/api/v1/auth/validate",
	}

	for _, endpoint := range publicEndpoints {
		if strings.HasPrefix(path, endpoint) {
			return true
		}
	}

	return false
}
