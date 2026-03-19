package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
)

// FAQSearchTool 显式实现 tool.InvokableTool 接口
type FAQSearchTool struct {
	embedder embedding.Embedder
	repo     *es.FAQESRepo
}

type FAQSearchInput struct {
	Query string `json:"query" jsonschema_description:"用户遇到的问题描述或关键词" jsonschema:"required"`
	TopK  int    `json:"top_k" jsonschema_description:"返回最相关的结果数量，默认 3" jsonschema:",omitempty"`
}

func NewFAQSearchTool(embedder embedding.Embedder, repo *es.FAQESRepo) *FAQSearchTool {
	return &FAQSearchTool{
		embedder: embedder,
		repo:     repo,
	}
}

func (t *FAQSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params, err := utils.GoStruct2ParamsOneOf[FAQSearchInput]()
	if err != nil {
		return nil, err
	}

	return &schema.ToolInfo{
		Name: "search_faq",
		Desc: `用于检索 FAQ（常见问题）数据库。

适用场景：
- 用户遇到系统问题（如登录失败、加载异常）
- 希望查找历史已有解决方案

示例：
用户问题: "课表不加载怎么办"
调用:
{
  "query": "课表 不加载 加载失败",
  "top_k": 3
}

返回 FAQ 列表（问题+答案），用于辅助回答。`,
		ParamsOneOf: params,
	}, nil
}

// InvokableRun 是核心执行逻辑
func (t *FAQSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 1. 解析 LLM 传入的 JSON 参数
	var input FAQSearchInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if input.TopK <= 0 {
		input.TopK = 3
	}

	// 2. 调用 Embedding 组件向量化
	vectors, err := t.embedder.EmbedStrings(ctx, []string{input.Query})
	if err != nil {
		return "", fmt.Errorf("embedding failed: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return "", fmt.Errorf("embedding returned empty vector")
	}

	// 3. 调用 ES Repo 进行检索
	hits, err := t.repo.SearchSimilarFAQ(ctx, vectors[0], input.TopK)
	if err != nil {
		return "", fmt.Errorf("es search failed: %w", err)
	}

	// 4. structured return
	if len(hits) == 0 {
		return `{"faq_results": []}`, nil
	}

	result := map[string]any{
		"faq_results": hits,
	}

	resp, _ := json.Marshal(result)
	return string(resp), nil
}
