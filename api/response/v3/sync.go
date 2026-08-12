package v3

// SyncRecordsResp 同步任务入队结果。
type SyncRecordsResp struct {
	RecordIDs []string `json:"record_ids"`
	QueueFull bool     `json:"queue_full"`
	Total     int      `json:"total"`
}
