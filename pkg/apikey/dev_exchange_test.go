//go:build devexchange

package apikey

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestGenerateLocalExchangeRequest 生成可直接用于本地调用 V3 Token Exchange 接口的请求体。
//
// 该测试只在 -tags devexchange 时编译，避免开发 API Key 出现在常规测试流程中。
// API Key 仅从环境变量读取，且不会输出到终端。
//
// 示例：
//
//	FEEDBACK_V3_PROJECT_ID=project-xxx \
//	FEEDBACK_V3_KEY_ID=project-xxx-key \
//	FEEDBACK_V3_API_KEY='...' \
//	FEEDBACK_V3_STUDENT_ID=2024214815 \
//	go test -count=1 -tags devexchange -run '^TestGenerateLocalExchangeRequest$' -v ./pkg/apikey
//
// 必须传入 -count=1 禁用 go test 缓存；否则可能复用上一次生成的 nonce，
// 被服务端的重放保护机制拒绝。
func TestGenerateLocalExchangeRequest(t *testing.T) {
	projectID := requiredEnv(t, "FEEDBACK_V3_PROJECT_ID")
	keyID := requiredEnv(t, "FEEDBACK_V3_KEY_ID")
	apiKey := requiredEnv(t, "FEEDBACK_V3_API_KEY")
	studentID := requiredEnv(t, "FEEDBACK_V3_STUDENT_ID")

	timestamp := time.Now().Unix()
	nonce := uuid.NewString()
	payload := strings.Join([]string{
		projectID,
		keyID,
		studentID,
		strconv.FormatInt(timestamp, 10),
		nonce,
	}, "\n")

	request := struct {
		ProjectID string `json:"project_id"`
		KeyID     string `json:"key_id"`
		StudentID string `json:"student_id"`
		Timestamp int64  `json:"timestamp"`
		Nonce     string `json:"nonce"`
		Signature string `json:"signature"`
	}{
		ProjectID: projectID,
		KeyID:     keyID,
		StudentID: studentID,
		Timestamp: timestamp,
		Nonce:     nonce,
		Signature: Sign(apiKey, payload),
	}

	body, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		t.Fatalf("序列化交换请求失败: %v", err)
	}

	endpoint := os.Getenv("FEEDBACK_V3_EXCHANGE_URL")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8080/api/v3/integrations/token/exchange"
	}
	t.Logf("将下面请求体发送到 %s：\n%s", endpoint, body)
	t.Log("请求在服务端的默认有效时间窗口为 5 分钟；使用 -count=1 执行时会生成新的 timestamp、nonce 和 signature。")
}

func requiredEnv(t *testing.T, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		t.Fatalf("缺少环境变量 %s", key)
	}
	return value
}
