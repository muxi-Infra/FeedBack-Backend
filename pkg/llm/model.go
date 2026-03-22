package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/philippgille/chromem-go"
)

// Config MiniMax (或任何兼容 OpenAI 格式的 LLM) 的配置结构体
type Config struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

// NewChatModel 构造一个遵循 Eino 标准接口的 ChatModel
func NewChatModel(ctx context.Context, cfg *Config) (model.ToolCallingChatModel, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("llm config: APIKey is required")
	}

	m, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init model: %w", err)
	}

	return m, nil
}

type LocalPureGoEmbedder struct {
	embFunc chromem.EmbeddingFunc
}

// NewLocalPureGoEmbedder 初始化本地纯 Go 向量化组件
func NewLocalPureGoEmbedder(ctx context.Context, baseUrl string) (*LocalPureGoEmbedder, error) {
	normalized := true
	// 访问指定路径的服务并进行请求
	embFunc := chromem.NewEmbeddingFuncOpenAICompat(baseUrl, "", "", &normalized)

	// 验证模型是否能正常响应 (可选)
	_, err := embFunc(ctx, "test")
	if err != nil {
		return nil, fmt.Errorf("无法访问emb模型,请检查是否正常: %w", err)
	}

	return &LocalPureGoEmbedder{
		embFunc: embFunc,
	}, nil
}

// EmbedStrings 实现 Eino 接口，并完成 float32 到 float64 的类型转换
func (e *LocalPureGoEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	res := make([][]float64, len(texts))

	for i := range texts {
		// 调用 llamafile 封装好的向量化函数 (返回 [][]float32)
		// 注意：chromem 的 embFunc 接收 []string 返回 [][]float32
		embResults, err := e.embFunc(ctx, texts[i])
		if err != nil {
			return nil, fmt.Errorf("本地向量化失败 (index %d): %w", i, err)
		}

		if len(embResults) == 0 {
			return nil, fmt.Errorf("模型未返回向量数据 (index %d)", i)
		}

		// 执行类型转换: []float32 -> []float64
		f32Vec := embResults
		f64Vec := make([]float64, len(f32Vec))
		for j, v := range f32Vec {
			f64Vec[j] = float64(v)
		}

		res[i] = f64Vec
	}

	return res, nil
}

type NLIResult struct {
	IsValid    bool
	Status     string
	Entailment float64
}

// NLIClient 抽象接口，方便后续 Mock 测试
type NLIClient interface {
	Check(ctx context.Context, premise, hypothesis string) (*NLIResult, error)
}

// nLIClient 基于 HTTP 的具体实现
type nLIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPNLIClient(baseURL string) NLIClient {
	return &nLIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{
			//Timeout: 500 * time.Millisecond, //需要的话可以设置一下
		},
	}
}

func (c *nLIClient) Check(ctx context.Context, premise, hypothesis string) (*NLIResult, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"premise":    premise,
		"hypothesis": hypothesis,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/nli/check", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nli backend unreachable: %w", err)
	}
	defer resp.Body.Close()

	var rawRes struct {
		IsValid bool   `json:"is_valid"`
		Status  string `json:"status"`
		Scores  struct {
			Entailment float64 `json:"entailment"`
		} `json:"scores"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawRes); err != nil {
		return nil, err
	}

	return &NLIResult{
		IsValid:    rawRes.IsValid,
		Status:     rawRes.Status,
		Entailment: rawRes.Scores.Entailment,
	}, nil
}
