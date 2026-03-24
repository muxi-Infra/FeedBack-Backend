package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/llm"
)

type FinalCheckTool struct {
	client llm.NLIClient
}

type FinalCheckInput struct {
	Answer     string `json:"answer" jsonschema_description:"根据文档得到的回答" jsonschema:"required"`
	DocContent string `json:"doc_content" jsonschema_description:"检索到的原始文档内容或FAQ答案" jsonschema:"required"`
}

func NewFinalCheckTool(client llm.NLIClient) *FinalCheckTool {
	return &FinalCheckTool{
		client: client,
	}
}

func (t *FinalCheckTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params, err := utils.GoStruct2ParamsOneOf[FinalCheckInput]()
	if err != nil {
		return nil, err
	}

	return &schema.ToolInfo{
		Name:        "final_check",
		Desc:        "逻辑校验工具。验证生成的 Answer 是否与 DocContent 保持一致，防止幻觉。",
		ParamsOneOf: params,
	}, nil
}

func (t *FinalCheckTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input FinalCheckInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// 调用解耦后的 Client
	nliRes, err := t.client.Check(ctx, input.DocContent, input.Answer)
	if err != nil {
		return "", err
	}

	finalResult := map[string]any{
		"logical_validity": nliRes.Status,
		"confidence":       nliRes.Entailment,
		"should_adopt":     nliRes.IsValid,
		"suggestion":       "If status is violation, DO NOT use this doc.",
	}

	resJSON, _ := json.Marshal(finalResult)
	return string(resJSON), nil
}
