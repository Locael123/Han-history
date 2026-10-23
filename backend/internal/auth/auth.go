// Package auth 负责签发与校验 JWT。
package auth

import (
	"errors"
	"time"

	"gonghe/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是自包含的登录态：校验时不查库。
type Claims struct {
	UserID   uint64 `json:"uid"`
	Username string `json:"usr"`
	jwt.RegisteredClaims
}

// Sign 签发 HS256 令牌。
func Sign(cfg *config.Config, userID uint64, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "gonghe",
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.JWTTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSecret))
}

// Parse 校验并解析令牌。
func Parse(cfg *config.Config, tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, errors.New("缺少令牌")
	}
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("令牌签名算法不匹配")
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, errors.New("登录已失效，请重新登录")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.UserID == 0 {
		return nil, errors.New("登录已失效，请重新登录")
	}
	return claims, nil
}
