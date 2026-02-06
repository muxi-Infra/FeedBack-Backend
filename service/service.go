package service

import (
	"sync"

	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
)

var ProviderSet = wire.NewSet(
	NewAuthService,
	NewSheetService,
	NewMessageService,
)

var (
	tableCfg   map[string]domain.TableConfig
	noticeCh   chan domain.TableConfig
	progressCh chan ProgressMsg
	once       sync.Once
)

type ProgressMsg struct {
	RecordID    string
	TableConfig domain.TableConfig
}

func init() {
	// 初始化通知通道，只执行一次
	once.Do(func() {
		tableCfg = make(map[string]domain.TableConfig)
		noticeCh = make(chan domain.TableConfig, 10)
		progressCh = make(chan ProgressMsg, 100)
	})
}

func simplifyFields(fields map[string]any) map[string]any {
	result := make(map[string]any, len(fields))

	for key, val := range fields {
		switch v := val.(type) {

		// 情况 1：[]any
		case []any:
			// 空数组
			if len(v) == 0 {
				result[key] = v
				continue
			}

			var fileTokens []string
			var text *string
			for _, item := range v {
				m, ok := item.(map[string]any)
				if !ok {
					break
				}

				// 文本字段
				if t, ok := m["text"].(string); ok {
					text = &t
					break
				}

				// 附件 / 图片字段（只要 file_token）
				if token, ok := m["file_token"].(string); ok {
					fileTokens = append(fileTokens, token)
					continue
				}
			}

			if text != nil {
				result[key] = *text
			} else if len(fileTokens) > 0 {
				result[key] = fileTokens
			} else {
				result[key] = v // 兜底
			}

		// 情况 2：已经是基础类型
		case string, float64, bool, int64, int:
			result[key] = v

		// 情况 3：其他未知结构
		// 尽量不要走到这一步，如果走到，即使增加情况处理
		default:
			result[key] = v
		}
	}

	return result
}
