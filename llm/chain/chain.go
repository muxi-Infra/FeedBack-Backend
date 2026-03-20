package chain

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/muxi-Infra/FeedBack-Backend/llm/prompts"
)

func NewSummaryChain(m model.ToolCallingChatModel) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
	chain := compose.NewChain[[]*schema.Message, *schema.Message]()
	tmpPrompt := prompt.FromMessages(schema.GoTemplate, schema.SystemMessage(prompts.SummarySystemPrompt), schema.UserMessage(prompts.SummaryUserPrompt))

	chain.
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, msgs []*schema.Message) (map[string]any, error) {
			return map[string]any{
				"history": FormatMessagesForSummary(msgs), // 上文提到的格式化函数
			}, nil
		})).
		AppendChatTemplate(tmpPrompt).
		AppendChatModel(m)
	compile, err := chain.Compile(context.Background())
	if err != nil {
		return nil, err
	}

	// 3. 调用 LLM
	return compile, nil
}

func FormatMessagesForSummary(msgs []*schema.Message) string {
	var builder strings.Builder
	for _, m := range msgs {
		roleName := string(m.Role)
		// 映射角色名，让 Prompt 更易理解
		if m.Role == schema.Assistant {
			roleName = "木小犀"
		} else if m.Role == schema.User {
			roleName = "用户"
		}

		builder.WriteString(fmt.Sprintf("%s: %s\n", roleName, m.Content))
	}
	return builder.String()
}
