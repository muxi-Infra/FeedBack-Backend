package llm

import (
	"context"
	"errors"
	"io"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/google/wire"
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
	faqRepo es.FAQESRepo,
	feedbackRepo es.FeedbackESRepo,
) *react.Agent {
	faqTool := tools.NewFAQSearchTool(embedder, &faqRepo)
	feedbackTool := tools.NewFeedbackSearchTool(embedder, &feedbackRepo)
	multiSearchTool := tools.NewMultiSearchTool(embedder, faqTool, feedbackTool)

	toolCallChecker := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
		defer sr.Close()
		for {
			msg, err := sr.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// finish
					break
				}

				return false, err
			}

			if len(msg.ToolCalls) > 0 {
				return true, nil
			}
		}
		return false, nil
	}

	//buildReact, err := agent.BuildReact(context.Background(), m,
	//	[]tool.BaseTool{
	//		faqTool,
	//		feedbackTool,
	//		multiSearchTool,
	//	},
	//	5, prompts.CustomerServicePersona)

	rAgent, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: m,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{faqTool, feedbackTool, multiSearchTool},
		},
		StreamToolCallChecker: toolCallChecker,
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			// 如果已经有 system message，就不要重复加
			if len(input) > 0 && input[0].Role == schema.System {
				return input
			}

			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(prompts.CustomerServicePersona))
			res = append(res, input...)
			return res
		},
	})
	if err != nil {
		panic(err)
		return nil
	}

	return rAgent
}
