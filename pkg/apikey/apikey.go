// Package apikey 提供项目 API Key 的生成和 HMAC 签名能力。
package apikey

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

// Generate 生成项目 API Key。明文只应在创建或轮换时返回给管理员一次。
func Generate() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// Digest 将 API Key 转换为服务端保存的摘要。
// HMAC 使用该摘要作为密钥，因此服务端不需要保存 API Key 明文。
func Digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// Sign 使用 API Key 为规范化请求内容生成 HMAC-SHA256 签名。
func Sign(value, payload string) string {
	mac := hmac.New(sha256.New, []byte(Digest(value)))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify 使用服务端保存的摘要验证 HMAC 签名。
func Verify(digest, payload, signature string) error {
	if digest == "" || signature == "" {
		return errors.New("api key signature is required")
	}
	mac := hmac.New(sha256.New, []byte(digest))
	_, _ = mac.Write([]byte(payload))
	expected := mac.Sum(nil)
	provided, err := hex.DecodeString(signature)
	if err != nil || len(provided) != len(expected) || subtle.ConstantTimeCompare(expected, provided) != 1 {
		return errors.New("invalid api key signature")
	}
	return nil
}
