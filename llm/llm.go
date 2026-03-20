package llm

import (
	"context"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/llm/agent"
	"github.com/muxi-Infra/FeedBack-Backend/llm/chain"
	"github.com/muxi-Infra/FeedBack-Backend/llm/prompts"
	"github.com/muxi-Infra/FeedBack-Backend/llm/tools"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
)

var ProviderSet = wire.NewSet(
	NewCustomerServiceReact,
	chain.NewSummaryChain,
)

func NewCustomerServiceReact(
	m model.ToolCallingChatModel,
	embedder embedding.Embedder,
	repo es.FAQESRepo,
) *react.Agent {
	faqTool := tools.NewFAQSearchTool(embedder, &repo)

	buildReact, err := agent.BuildReact(context.Background(), m, []tool.BaseTool{faqTool}, 5, prompts.CustomerServicePersona)
	if err != nil {
		panic(err)
		return nil
	}

	return buildReact
}
