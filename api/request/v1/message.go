package v1

type TriggerNotificationReq struct {
	TableIdentify string `json:"table_identify" binding:"required"` // 反馈表格 Identify，反馈表的唯一标识
}

type MarkRecordNoticedReq struct {
	TableIdentify string `json:"table_identify" binding:"required"` // 反馈表格 Identify，反馈表的唯一标识
	RecordID      string `json:"record_id" binding:"required"`      // 反馈记录 ID，记录的唯一标识
}
