package request

type CreateAppReq struct {
	Name        string `json:"name" binding:"required"`
	FolderToken string `json:"folder_token,omitempty"` // 可以为空，但是要有
}

type CopyAppReq struct {
	AppToken       string `json:"app_token" binding:"required"` // 需要复制的表格的token
	Name           string `json:"name" binding:"required"`
	FolderToken    string `json:"folder_token,omitempty"`       // 可以为空，但是要有
	WithoutContent bool   `json:"without_content"`              // 是否复制内容
	TimeZone       string `json:"time_zone" binding:"required"` //时区,例如: Asia/Shanghai
}

type CreateAppTableRecordReq struct {
	AppToken string `json:"app_token" binding:"required"`
	TableId  string `json:"table_id" binding:"required"`
	//UserIdType             string                 `json:"user_id_type,omitempty"`
	//ClientToken            string                 `json:"client_token,omitempty"`
	IgnoreConsistencyCheck bool                   `json:"ignore_consistency_check,omitempty"`
	Fields                 map[string]interface{} `json:"fields" binding:"required"` // 记录的字段
}
