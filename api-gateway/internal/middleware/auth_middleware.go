package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amirhasanpour/task-manager/api-gateway/internal/client"
	pb "github.com/amirhasanpour/task-manager/api-gateway/proto"
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

		// Validate token
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

func (m *AuthMiddleware) validateToken(ctx context.Context, tokenString string) (*pb.User, error) {
	// First, try to parse and validate JWT locally for performance
	claims, err := m.parseJWT(tokenString)
	if err != nil {
		m.logger.Debug("Failed to parse JWT locally", zap.Error(err))
		// Fall back to user service validation
		return m.validateWithUserService(ctx, tokenString)
	}

	// Extract user info from claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		m.logger.Debug("Missing user_id in JWT claims")
		return m.validateWithUserService(ctx, tokenString)
	}

	// Return user info from claims
	return &pb.User{
		Id:       userID,
		Username: m.getStringFromClaims(claims, "username"),
		Email:    m.getStringFromClaims(claims, "email"),
		FullName: m.getStringFromClaims(claims, "full_name"),
	}, nil
}

func (m *AuthMiddleware) parseJWT(tokenString string) (jwt.MapClaims, error) {
	m.logger.Debug("Attempting to parse JWT token", 
		zap.String("token_prefix", tokenString[:min(50, len(tokenString))]),
	)
	
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		m.logger.Error("JWT parse error", 
			zap.Error(err),
			zap.Any("token_header", token.Header),
		)
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		m.logger.Error("JWT token invalid")
		return nil, errors.New("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		m.logger.Error("Invalid JWT claims type")
		return nil, errors.New("invalid token claims")
	}

	// Debug: Print all claims
	m.logger.Debug("JWT claims", zap.Any("claims", claims))
	
	// Check expiration
	exp, ok := claims["exp"].(float64)
	if ok {
		expTime := time.Unix(int64(exp), 0)
		m.logger.Debug("Token expiration", 
			zap.Time("expiration_time", expTime),
			zap.Time("current_time", time.Now()),
			zap.Duration("time_until_expiry", time.Until(expTime)),
		)
		
		if time.Now().After(expTime) {
			m.logger.Error("Token expired", 
				zap.Time("expired_at", expTime),
				zap.Duration("expired_by", time.Since(expTime)),
			)
			return nil, errors.New("token has expired")
		}
	} else {
		m.logger.Warn("No expiration claim in token")
	}

	// Check if token is issued at valid time
	if iat, ok := claims["iat"].(float64); ok {
		iatTime := time.Unix(int64(iat), 0)
		if time.Now().Before(iatTime) {
			m.logger.Error("Token issued in future", 
				zap.Time("issued_at", iatTime),
				zap.Duration("time_until_issue", time.Until(iatTime)),
			)
			return nil, errors.New("token issued at invalid time")
		}
	}

	return claims, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *AuthMiddleware) getStringFromClaims(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (m *AuthMiddleware) validateWithUserService(ctx context.Context, tokenString string) (*pb.User, error) {
	req := &pb.ValidateTokenRequest{
		Token: tokenString,
	}

	resp, err := m.userClient.ValidateToken(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
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