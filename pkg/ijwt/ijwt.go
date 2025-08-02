package ijwt

import (
	"errors"
	"feedback/config"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type JWT struct {
	signingMethod jwt.SigningMethod // JWT 签名方法
	rcExpiration  time.Duration     // 刷新令牌的过期时间，防止缓存过大
	jwtKey        []byte            // 用于签署 JWT 的密钥
}

func NewJWT(conf config.JWTConfig) *JWT {
	return &JWT{
		signingMethod: jwt.SigningMethodHS256, //签名的加密方式
		rcExpiration:  time.Duration(conf.Timeout) * time.Second,
		jwtKey:        []byte(conf.SecretKey),
	}
}

type UserClaims struct {
	jwt.RegisteredClaims // 内嵌标准的声明
	//Token                string `json:"token"` // 飞书令牌
	TableID       string `json:"table_id"`
	NormalTableID string `json:"normal_table_id"` // 常见问题表ID
}

func (j *JWT) SetJWTToken(t, nt string) (string, error) {
	uc := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.rcExpiration)),
		},
		TableID:       t,
		NormalTableID: nt,
	}

	token := jwt.NewWithClaims(j.signingMethod, uc)
	// 使用指定的secret签名并获得完整的编码后的字符串token
	tokenStr, err := token.SignedString(j.jwtKey)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

// ParseToken 从请求中提取并返回解析完成的结构体
func (j *JWT) ParseToken(tokenStr string) (UserClaims, error) {
	if j.jwtKey == nil {
		return UserClaims{}, errors.New("jwtKey 为空，无法解析 token")
	}

	//解析token
	uc := UserClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, &uc, func(token *jwt.Token) (interface{}, error) {
		// 校验签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("签名检验算法错误")
		}
		return j.jwtKey, nil
	})
	if err != nil {
		return UserClaims{}, err
	}

	//检查有效性
	if token == nil || !token.Valid {
		return UserClaims{}, errors.New("token无效")
	}

	// 确保 Claims 解析成功
	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return UserClaims{}, errors.New("无法解析 token claims")
	}

	return *claims, nil
}
