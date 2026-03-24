package llm

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/llm/chain"
	"github.com/muxi-Infra/FeedBack-Backend/llm/prompts"
	"github.com/muxi-Infra/FeedBack-Backend/llm/skills"
	"github.com/muxi-Infra/FeedBack-Backend/llm/tools"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/llm"
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
	nLIClient llm.NLIClient,
) *adk.Runner {
	ctx := context.Background()
	faqTool := tools.NewFAQSearchTool(embedder, &faqRepo)
	feedbackTool := tools.NewFeedbackSearchTool(embedder, &feedbackRepo)
	multiSearchTool := tools.NewMultiSearchTool(embedder, faqTool, feedbackTool)
	finalCheckTool := tools.NewFinalCheckTool(nLIClient)

	pwd, _ := os.Getwd()

	skillsDir := filepath.Join(pwd, "llm", "skills")

	skillBackend, err := skills.LoadSkills(ctx, skillsDir)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	agent, err := deep.New(ctx, &deep.Config{
		Name:        "Customer Agent",
		Description: "用于处理木犀反馈的客服机器人",
		ChatModel:   m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{faqTool, feedbackTool, multiSearchTool, finalCheckTool},
			},
		},
		MaxIteration: 5,
		Middlewares: []adk.AgentMiddleware{
			{
				BeforeChatModel: func(ctx context.Context, input *adk.ChatModelAgentState) error {
					// 如果已经有 system message，就不要重复加
					if len(input.Messages) > 0 && input.Messages[0].Role == schema.System {
						return nil
					}
					input.Messages = append(input.Messages, schema.SystemMessage(prompts.CustomerServicePersona))
					return nil
				},
			},
		},
		Handlers: []adk.ChatModelAgentMiddleware{skillMiddleware},
	})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		//CheckPointStore: , // 后续如果需要做自动反馈的能力的话可以使用这个字段实现断点状态存储,从而获取多轮对话命中一个skill的多个阶段的效果
	})

	return runner
}

//func NewCustomerServiceReact(
//	m model.ToolCallingChatModel,
//	embedder embedding.Embedder,
//	faqRepo es.FAQESRepo,
//	feedbackRepo es.FeedbackESRepo,
//	nLIClient llm.NLIClient,
//) *react.Agent {
//	faqTool := tools.NewFAQSearchTool(embedder, &faqRepo)
//	feedbackTool := tools.NewFeedbackSearchTool(embedder, &feedbackRepo)
//	multiSearchTool := tools.NewMultiSearchTool(embedder, faqTool, feedbackTool)
//	finalCheckTool := tools.NewFinalCheckTool(nLIClient)
//	toolCallChecker := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
//		defer sr.Close()
//		for {
//			msg, err := sr.Recv()
//			if err != nil {
//				if errors.Is(err, io.EOF) {
//					// finish
//					break
//				}
//
//				return false, err
//			}
//
//			if len(msg.ToolCalls) > 0 {
//				return true, nil
//			}
//		}
//		return false, nil
//	}
//
//	rAgent, err := react.NewAgent(context.Background(), &react.AgentConfig{
//		ToolCallingModel: m,
//		ToolsConfig: compose.ToolsNodeConfig{
//			Tools: []tool.BaseTool{faqTool, feedbackTool, multiSearchTool, finalCheckTool},
//		},
//		StreamToolCallChecker: toolCallChecker,
//		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
//			// 如果已经有 system message，就不要重复加
//			if len(input) > 0 && input[0].Role == schema.System {
//				return input
//			}
//
//			res := make([]*schema.Message, 0, len(input)+1)
//			res = append(res, schema.SystemMessage(prompts.CustomerServicePersona))
//			res = append(res, input...)
//			return res
//		},
//	})
//	if err != nil {
//		panic(err)
//		return nil
//	}
//
//	return rAgent
//}
