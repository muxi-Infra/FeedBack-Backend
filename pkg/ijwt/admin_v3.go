package ijwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muxi-Infra/FeedBack-Backend/config"
)

type AdminClaimsV3 struct {
	jwt.RegisteredClaims
	TokenType string `json:"type"` // 用于标识管理员 Token
}

type AdminJWTV3 struct {
	key      []byte
	issuer   string
	audience string
	ttl      time.Duration
}

func NewAdminJWTV3(cfg config.AdminJWTConfig) *AdminJWTV3 {
	return &AdminJWTV3{
		key:      []byte(cfg.SecretKey),
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		ttl:      time.Duration(cfg.Timeout) * time.Second,
	}
}

func (j *AdminJWTV3) Issue(adminID uint64) (string, error) {
	if j == nil || len(j.key) == 0 || adminID == 0 {
		return "", errors.New("invalid admin jwt arguments")
	}
	now := time.Now()
	claims := AdminClaimsV3{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: fmt.Sprintf("%d", adminID),
			Issuer:  j.issuer,
			Audience: jwt.ClaimStrings{
				j.audience,
			},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
		TokenType: "admin",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.key)
}

func (j *AdminJWTV3) Parse(value string) (AdminClaimsV3, error) {
	if j == nil || len(j.key) == 0 || strings.TrimSpace(value) == "" {
		return AdminClaimsV3{}, errors.New("invalid admin jwt")
	}
	claims := &AdminClaimsV3{}
	token, err := jwt.ParseWithClaims(value, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid admin jwt algorithm")
		}
		return j.key, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(j.issuer), jwt.WithAudience(j.audience))
	if err != nil || token == nil || !token.Valid || claims.TokenType != "admin" {
		return AdminClaimsV3{}, errors.New("invalid or expired admin jwt")
	}
	return *claims, nil
}

func (j *AdminJWTV3) TTLSeconds() int64 {
	return int64(j.ttl / time.Second)
}
