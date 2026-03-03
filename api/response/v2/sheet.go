package v2

import "github.com/muxi-Infra/FeedBack-Backend/domain"

// GetTableRecordResp 获取表格记录返回参数（个人历史记录）
type GetTableRecordResp struct {
	Records   []domain.TableRecord `json:"records"`
	HasMore   bool                 `json:"has_more"`
	PageToken string               `json:"page_token"`
}

// SyncUnsyncedTableRecordsResp 同步指定表格下所有未同步的记录请求参数（不区分用户））
type SyncUnsyncedTableRecordsResp struct {
	RecordIDs []string `json:"record_ids"` // 成功投递到队列的记录 ID
	QueueFull bool     `json:"queue_full"` // 队列是否已满
	Total     int      `json:"total"`      // 本次尝试同步的总记录数
}

type ForceSyncUserTableRecordsResp struct {
	RecordIDs []string `json:"record_ids"` // 成功投递到队列的记录 ID
	QueueFull bool     `json:"queue_full"` // 队列是否已满
	Total     int      `json:"total"`      // 本次尝试同步的总记录数
}
