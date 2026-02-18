package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT声明
type Claims struct {
	UserID     string `json:"userId"`
	DeviceID   string `json:"deviceId"`
	DeviceType string `json:"deviceType"`
	TokenType  string `json:"tokenType"` // access, refresh
	jwt.RegisteredClaims
}

// Config JWT配置
type Config struct {
	Secret             string
	AccessTokenExpire  time.Duration
	RefreshTokenExpire time.Duration
}

// Manager JWT管理器
type Manager struct {
	config *Config
}

// NewManager 创建JWT管理器
func NewManager(config *Config) *Manager {
	return &Manager{
		config: config,
	}
}

// GenerateAccessToken 生成AccessToken
func (m *Manager) GenerateAccessToken(userID, deviceID, deviceType string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceType: deviceType,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.AccessTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// GenerateRefreshToken 生成RefreshToken
func (m *Manager) GenerateRefreshToken(userID, deviceID, deviceType string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceType: deviceType,
		TokenType:  "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.RefreshTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// ParseToken 解析Token
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateAccessToken 验证AccessToken
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := m.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}

// ValidateRefreshToken 验证RefreshToken
func (m *Manager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}
