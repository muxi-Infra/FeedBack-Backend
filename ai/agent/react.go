package agent

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// BuildReact 这里使用react包自动完成第一版的agent,之后可以考虑使用eino/compose手动去做React的编排
func BuildReact(
	ctx context.Context,
	m model.ToolCallingChatModel,
	tools []tool.BaseTool,
	maxStep int,
	systemPrompt string,
) (*react.Agent, error) {

	// 创建 agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: m,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: maxStep,
		//这个相当于一个拦截器/中间件,会把拦截到的消息进行处理,这里等于在最前面注入了一个system prompt
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(systemPrompt))
			res = append(res, input...)
			return res
		},
	})
	if err != nil {
		return nil, err
	}

	return agent, nil
}
