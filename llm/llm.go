package llm

import (
	"context"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/llm/agent"
	"github.com/muxi-Infra/FeedBack-Backend/llm/prompts"
	"github.com/muxi-Infra/FeedBack-Backend/llm/tools"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
)

var ProviderSet = wire.NewSet(NewCustomerServiceReact)

func NewCustomerServiceReact(
	m model.ToolCallingChatModel,
	embedder embedding.Embedder,
	faqRepo es.FAQESRepo,
	feedbackRepo es.FeedbackESRepo,
) *react.Agent {
	faqTool := tools.NewFAQSearchTool(embedder, &faqRepo)
	feedbackTool := tools.NewFeedbackSearchTool(embedder, &feedbackRepo)
	multiSearchTool := tools.NewMultiSearchTool(embedder, faqTool, feedbackTool)

	buildReact, err := agent.BuildReact(context.Background(), m,
		[]tool.BaseTool{
			faqTool,
			feedbackTool,
			multiSearchTool,
		},
		5, prompts.CustomerServicePersona)
	if err != nil {
		panic(err)
		return nil
	}

	return buildReact
}
