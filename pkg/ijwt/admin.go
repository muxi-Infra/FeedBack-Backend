package ijwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muxi-Infra/FeedBack-Backend/config"
)

const adminTokenType = "admin"

// AdminClaims 管理后台 JWT 的声明。
// 内嵌 RegisteredClaims.Subject（JWT 的 sub 字段）约定存储管理员 ID。
// 管理员角色和权限不放入 Token，统一由 Casbin 实时判断。
type AdminClaims struct {
	jwt.RegisteredClaims
	TokenType string `json:"type"`
}

type AdminJWT struct {
	secretKey []byte
	issuer    string
	audience  string
	timeout   time.Duration
}

func NewAdminJWT(cfg config.AdminJWTConfig) *AdminJWT {
	return &AdminJWT{
		secretKey: []byte(cfg.SecretKey),
		issuer:    cfg.Issuer,
		audience:  cfg.Audience,
		timeout:   time.Duration(cfg.Timeout) * time.Second,
	}
}

// Issue 为管理员签发短期访问令牌。
func (j *AdminJWT) Issue(subject uint64) (string, time.Time, error) {
	if j == nil || len(j.secretKey) == 0 {
		return "", time.Time{}, errors.New("admin jwt secret key is empty")
	}
	if subject == 0 {
		return "", time.Time{}, errors.New("admin jwt subject is required")
	}
	if j.timeout <= 0 {
		return "", time.Time{}, errors.New("admin jwt timeout must be positive")
	}

	now := time.Now()
	expiresAt := now.Add(j.timeout)
	claims := AdminClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", subject),
			Issuer:    j.issuer,
			Audience:  jwt.ClaimStrings{j.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		TokenType: adminTokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// Parse 校验并解析管理员 JWT。
func (j *AdminJWT) Parse(tokenString string) (AdminClaims, error) {
	if j == nil || len(j.secretKey) == 0 {
		return AdminClaims{}, errors.New("admin jwt secret key is empty")
	}
	if strings.TrimSpace(tokenString) == "" {
		return AdminClaims{}, errors.New("admin jwt token is empty")
	}

	claims := &AdminClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return j.secretKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(j.issuer), jwt.WithAudience(j.audience))
	if err != nil {
		return AdminClaims{}, err
	}
	if token == nil || !token.Valid {
		return AdminClaims{}, errors.New("admin jwt token is invalid")
	}
	if claims.TokenType != adminTokenType {
		return AdminClaims{}, errors.New("token type is not admin")
	}
	return *claims, nil
}
