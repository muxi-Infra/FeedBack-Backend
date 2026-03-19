package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type MultiSearchTool struct {
	embedder embedding.Embedder

	faqTool      *FAQSearchTool
	feedbackTool *FeedbackSearchTool
}

type MultiSearchInput struct {
	Query string `json:"query" jsonschema_description:"用户问题关键词" jsonschema:"required"`
	TopK  int    `json:"top_k" jsonschema_description:"每个数据源返回数量，默认 3" jsonschema:",omitempty"`
}

func NewMultiSearchTool(
	embedder embedding.Embedder,
	faqTool *FAQSearchTool,
	feedbackTool *FeedbackSearchTool,
) *MultiSearchTool {
	return &MultiSearchTool{
		embedder:     embedder,
		faqTool:      faqTool,
		feedbackTool: feedbackTool,
	}
}

func (t *MultiSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params, err := utils.GoStruct2ParamsOneOf[MultiSearchInput]()
	if err != nil {
		return nil, err
	}

	return &schema.ToolInfo{
		Name: "multi_search",
		Desc: `用于同时检索 FAQ 和用户反馈记录。

适用场景：
- 用户遇到系统问题（报错、无法使用等）
- 需要结合“标准答案 + 实际案例”进行回答

内部会自动：
1. 查询 FAQ
2. 查询用户反馈记录
3. 合并返回结果`,
		ParamsOneOf: params,
	}, nil
}

func (t *MultiSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 1️⃣ 解析参数
	var input MultiSearchInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if input.TopK <= 0 {
		input.TopK = 3
	}

	// 2️⃣ embedding（只做一次）
	vectors, err := t.embedder.EmbedStrings(ctx, []string{input.Query})
	if err != nil {
		return "", fmt.Errorf("embedding failed: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return "", fmt.Errorf("embedding returned empty vector")
	}

	queryVec := vectors[0]

	// 3️⃣ 并发查询
	var wg sync.WaitGroup
	wg.Add(2)

	var faqResults []any
	var feedbackResults []any

	var faqErr error
	var feedbackErr error

	// FAQ 查询
	go func() {
		defer wg.Done()
		hits, err := t.faqTool.repo.SearchSimilarFAQ(ctx, queryVec, input.TopK)
		if err != nil {
			faqErr = err
			return
		}
		for _, h := range hits {
			faqResults = append(faqResults, h)
		}
	}()

	// Feedback 查询
	go func() {
		defer wg.Done()
		hits, err := t.feedbackTool.repo.SearchSimilarFeedback(ctx, queryVec, input.TopK)
		if err != nil {
			feedbackErr = err
			return
		}
		for _, h := range hits {
			feedbackResults = append(feedbackResults, h)
		}
	}()

	wg.Wait()

	// 4️⃣ 错误处理（容忍部分失败）
	if faqErr != nil && feedbackErr != nil {
		return "", fmt.Errorf("both searches failed: faq=%v, feedback=%v", faqErr, feedbackErr)
	}

	// 5️⃣ 统一返回结构
	result := map[string]any{
		"faq_results":      faqResults,
		"feedback_results": feedbackResults,
	}

	resp, _ := json.Marshal(result)
	return string(resp), nil
}
