// Package auth 负责密码哈希、会话令牌签发与校验，为鉴权中间件提供统一入口。
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	issuer       = "veristack"
	tokenSubject = "access"
)

// Claims 描述会话令牌携带的身份信息。
type Claims struct {
	UserID string // 用户 ID
	Role   string // 用户角色
	jwt.RegisteredClaims
}

// Manager 负责令牌的签发与解析，Secret 为 HMAC 签名密钥。
type Manager struct {
	ExpiresIn time.Duration // 令牌有效期
	Secret    []byte        // 签名密钥
}

// NewManager 创建令牌管理器。
func NewManager(secret string, expiresIn time.Duration) *Manager {
	return &Manager{ExpiresIn: expiresIn, Secret: []byte(secret)}
}

// Generate 为指定用户签发访问令牌。
func (m *Manager) Generate(userID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    issuer,
			Subject:   tokenSubject,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.Secret)
}

// Parse 校验令牌签名与有效期，返回其中的身份声明。
func (m *Manager) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("不支持的令牌签名算法: %s", t.Method.Alg())
		}
		return m.Secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("解析令牌失败: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("令牌无效")
	}
	return claims, nil
}
