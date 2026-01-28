package request

type TriggerNotificationReq struct {
	TableIdentify string `json:"table_identify" binding:"required"` // 反馈表格 Identify，反馈表的唯一标识
}
