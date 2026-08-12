package ijwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muxi-Infra/FeedBack-Backend/config"
)

type V3UserClaims struct {
	jwt.RegisteredClaims
	ProjectID string `json:"project_id"`
	StudentID string `json:"student_id"`
}

type V3JWT struct {
	key []byte
	ttl time.Duration
}

func NewV3JWT(conf config.JWTConfig, auth *config.IntegrationAuthConfig) *V3JWT {
	ttl := time.Duration(conf.Timeout) * time.Second
	if auth != nil && auth.AccessTokenTTL > 0 {
		ttl = time.Duration(auth.AccessTokenTTL) * time.Second
	}
	return &V3JWT{
		key: []byte(conf.SecretKey),
		ttl: ttl,
	}
}

func (j *V3JWT) Issue(projectID, studentID string) (string, int64, error) {
	if len(j.key) == 0 || projectID == "" || studentID == "" {
		return "", 0, errors.New("invalid v3 jwt arguments")
	}
	expires := time.Now().Add(j.ttl)
	claims := V3UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		ProjectID: projectID,
		StudentID: studentID,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.key)
	if err != nil {
		return "", 0, err
	}
	return token, int64(time.Until(expires).Seconds()), nil
}

func (j *V3JWT) Parse(tokenString string) (V3UserClaims, error) {
	var claims V3UserClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid v3 jwt signing algorithm")
		}
		return j.key, nil
	})
	if err != nil || token == nil || !token.Valid {
		return V3UserClaims{}, errors.New("invalid or expired v3 jwt")
	}
	if claims.ProjectID == "" || claims.StudentID == "" {
		return V3UserClaims{}, errors.New("v3 jwt identity is missing")
	}
	return claims, nil
}
