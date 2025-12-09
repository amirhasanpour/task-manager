package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/amirhasanpour/task-manager/user-service/internal/model"
	"go.uber.org/zap"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
	logger        *zap.Logger
}

func NewJWTManager(secretKey string, tokenDurationHours int) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: time.Duration(tokenDurationHours) * time.Hour,
		logger:        zap.L().Named("jwt_manager"),
	}
}

func (manager *JWTManager) Generate(user *model.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(manager.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(manager.secretKey))
	if err != nil {
		manager.logger.Error("Failed to generate token", zap.Error(err))
		return "", err
	}

	manager.logger.Debug("Token generated successfully", zap.String("user_id", user.ID))
	return tokenString, nil
}

func (manager *JWTManager) Verify(accessToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		accessToken,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, errors.New("unexpected token signing method")
			}
			return []byte(manager.secretKey), nil
		},
	)

	if err != nil {
		manager.logger.Error("Failed to parse token", zap.Error(err))
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		manager.logger.Error("Invalid token claims")
		return nil, errors.New("invalid token claims")
	}

	if !token.Valid {
		manager.logger.Error("Invalid token")
		return nil, errors.New("invalid token")
	}

	manager.logger.Debug("Token verified successfully", zap.String("user_id", claims.UserID))
	return claims, nil
}

func (manager *JWTManager) Validate(accessToken string) (bool, *Claims) {
	claims, err := manager.Verify(accessToken)
	if err != nil {
		manager.logger.Error("Token validation failed", zap.Error(err))
		return false, nil
	}
	return true, claims
}