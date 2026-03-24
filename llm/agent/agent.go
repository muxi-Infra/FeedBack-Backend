package agent

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/muxi-Infra/FeedBack-Backend/llm/skills"
)

// 暂时弃用
// BuildReact 这里使用react包自动完成第一版的agent,之后可以考虑使用eino/compose手动去做React的编排
func BuildReact(ctx context.Context, m model.ToolCallingChatModel, tools []tool.BaseTool, maxStep int, systemPrompt string) (*adk.Runner, error) {
	pwd, _ := os.Getwd()

	skillsDir := filepath.Join(pwd, "llm", "skills")

	skillBackend, err := skills.LoadSkills(ctx, skillsDir)
	if err != nil {
		log.Fatal(err)
	}

	// 合并 system prompt + skills
	finalSystemPrompt := systemPrompt + `

# Tool Usage Rules（必须遵守）

你必须严格遵守以下规则：

1. 当 Skill 中标注“必须调用工具”时：
   - 不允许直接回答
   - 必须先调用真实工具
   - 禁止自行编造工具返回结果

2. 严禁出现以下行为：
   - 假装调用工具
   - 直接生成“工具返回的数据结构”
   - 在没有调用工具的情况下说“根据工具结果”

3. 如果没有调用工具：
   - 不允许提及“工具结果”

4. 工具调用是唯一的数据来源：
   - 不能用训练知识替代

`
	skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		log.Fatal(err)
	}

	agent, err := deep.New(ctx, &deep.Config{
		ChatModel: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIteration: maxStep,
		Middlewares: []adk.AgentMiddleware{
			{
				BeforeChatModel: func(ctx context.Context, input *adk.ChatModelAgentState) error {
					// 如果已经有 system message，就不要重复加
					if len(input.Messages) > 0 && input.Messages[0].Role == schema.System {
						return nil
					}
					input.Messages = append(input.Messages, schema.SystemMessage(finalSystemPrompt))
					return nil
				},
			},
		},
		Handlers: []adk.ChatModelAgentMiddleware{skillMiddleware},
	})
	if err != nil {
		return nil, err
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		//CheckPointStore: , // 后续如果需要做自动反馈的能力的话可以使用这个字段实现断点状态存储,从而获取多轮对话命中一个skill的多个阶段的效果
	})

	return runner, nil
}
