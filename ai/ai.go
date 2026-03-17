package ai

import (
	"context"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/ai/agent"
	"github.com/muxi-Infra/FeedBack-Backend/ai/prompts"
	"github.com/muxi-Infra/FeedBack-Backend/ai/tools"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
)

var ProviderSet = wire.NewSet(NewCustomerServiceReact)

func NewCustomerServiceReact(
	m model.ToolCallingChatModel,
	embedder embedding.Embedder,
	repo *es.FAQESRepo,
) *react.Agent {

	faqTool := tools.NewFAQSearchTool(embedder, repo)

	buildReact, err := agent.BuildReact(context.Background(), m, []tool.BaseTool{faqTool}, 5, prompts.CustomerServicePersona)
	if err != nil {
		panic(err)
		return nil
	}

	return buildReact
}
