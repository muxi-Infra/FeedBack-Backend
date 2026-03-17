package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
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

// Info 定义工具的元数据，供 LLM 识别
func (t *FAQSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params, err := utils.GoStruct2ParamsOneOf[FAQSearchInput]()
	if err != nil {
		return nil, err
	}

	return &schema.ToolInfo{
		Name: "faq_search",
		Desc: "搜索历史 FAQ 数据库。当你需要查询是否有类似问题的解决方案、" +
			"查看问题反馈频率或确认问题是否已解决时调用。",
		// 参数描述，帮助 LLM 生成正确的 JSON
		ParamsOneOf: params,
	}, nil
}

// InvokableRun 是核心执行逻辑
func (t *FAQSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...any) (string, error) {
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
	if len(vectors) == 0 {
		return "未搜索到相关信息", nil
	}

	// 3. 调用你的 ES Repo 进行检索
	hits, err := t.repo.SearchSimilarFAQ(ctx, vectors[0], input.TopK)
	if err != nil {
		return "", fmt.Errorf("es search failed: %w", err)
	}

	// 4. 将结果转换为字符串返回给 LLM
	// 注意：返回给大模型的内容最好是清晰、精简的 JSON 或文本
	if len(hits) == 0 {
		return "在 FAQ 数据库中没有找到相关的历史记录。", nil
	}

	resp, _ := json.Marshal(hits)
	return string(resp), nil
}
