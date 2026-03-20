package agent

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/muxi-Infra/FeedBack-Backend/llm/skills"
)

// 暂时弃用
// BuildReact 这里使用react包自动完成第一版的agent,之后可以考虑使用eino/compose手动去做React的编排
func BuildReact(ctx context.Context, m model.ToolCallingChatModel, tools []tool.BaseTool, maxStep int, systemPrompt string) (*react.Agent, error) {
	pwd, _ := os.Getwd()

	skillsDir := filepath.Join(pwd, "llm", "skills")

	skillPrompt, err := skills.LoadSkills(skillsDir)
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

` + "\n\n" + skillPrompt

	// 创建 agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: m,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: maxStep,
		//这个相当于一个拦截器/中间件,会把拦截到的消息进行处理,这里等于在最前面注入了一个system prompt
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			// 如果已经有 system message，就不要重复加
			if len(input) > 0 && input[0].Role == schema.System {
				return input
			}

			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(finalSystemPrompt))
			res = append(res, input...)
			return res
		},
	})
	if err != nil {
		return nil, err
	}

	return agent, nil
}
