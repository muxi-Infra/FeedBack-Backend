package message

// GroupMessageRequest 定义群消息请求结构体
type GroupMessageRequest struct {
	GroupID int64         `json:"group_id"`
	Message []MessageItem `json:"message"`
}

// MessageItem 定义消息项结构体
type MessageItem struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}
