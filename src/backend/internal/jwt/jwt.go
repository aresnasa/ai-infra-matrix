package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

type Claims struct {
	UserID      uint     `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`       // 用户的所有角色
	Permissions []string `json:"permissions"` // 用户的所有权限
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT token
func GenerateToken(userID uint, username string, roles []string, permissions []string) (string, int64, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // 24小时过期
	claims := &Claims{
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", 0, err
	}

	return tokenString, expirationTime.Unix(), nil
}

// ParseToken 解析JWT token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// SetSecret 设置JWT密钥
func SetSecret(secret string) {
	if secret != "" {
		jwtSecret = []byte(secret)
	}
}
