// Package apikey 提供 V3 项目 API Key 的生成和请求签名能力。
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

func Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func Digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func Sign(value, payload string) string {
	mac := hmac.New(sha256.New, []byte(Digest(value)))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

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
