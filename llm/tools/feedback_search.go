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

type FeedbackSearchTool struct {
	embedder embedding.Embedder
	repo     *es.FeedbackESRepo
}

type FeedbackSearchInput struct {
	Query string `json:"query" jsonschema_description:"用户遇到的问题描述或关键词" jsonschema:"required"`
	TopK  int    `json:"top_k" jsonschema_description:"返回最相关的结果数量，默认 3" jsonschema:",omitempty"`
}

func NewFeedbackSearchTool(embedder embedding.Embedder, repo *es.FeedbackESRepo) *FeedbackSearchTool {
	return &FeedbackSearchTool{
		embedder: embedder,
		repo:     repo,
	}
}

func (t *FeedbackSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params, err := utils.GoStruct2ParamsOneOf[FeedbackSearchInput]()
	if err != nil {
		return nil, err
	}

	return &schema.ToolInfo{
		Name: "search_feedback",
		Desc: `用于检索已完成的用户反馈记录（真实问题及解决过程）。

适用场景：
- 查询类似问题是否被用户反馈过
- 查看真实问题的解决方式
- FAQ 不足以覆盖时作为补充

示例：
用户问题: "提交作业时报500错误"
调用:
{
  "query": "提交作业 500错误 服务器错误",
  "top_k": 3
}

返回历史反馈记录（问题+处理结果）。`,
		ParamsOneOf: params,
	}, nil
}

func (t *FeedbackSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input FeedbackSearchInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if input.TopK <= 0 {
		input.TopK = 3
	}

	// embedding
	vectors, err := t.embedder.EmbedStrings(ctx, []string{input.Query})
	if err != nil {
		return "", fmt.Errorf("embedding failed: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return "", fmt.Errorf("embedding returned empty vector")
	}

	// search
	hits, err := t.repo.SearchSimilarFeedback(ctx, vectors[0], input.TopK)
	if err != nil {
		return "", fmt.Errorf("es search failed: %w", err)
	}

	// no result
	if len(hits) == 0 {
		return `{"feedback_results": []}`, nil
	}

	result := map[string]any{
		"feedback_results": hits,
	}

	resp, _ := json.Marshal(result)
	return string(resp), nil
}
