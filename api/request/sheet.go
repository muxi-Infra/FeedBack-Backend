package request

type CreateAppReq struct {
	Name        string `json:"name" binding:"required"` // 多维表格app名称
	FolderToken string `json:"folder_token,omitempty"`  // 可以为空，但是要有 多维表格app归属目录
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

type GetAppTableRecordReq struct {
	AppToken   string   `json:"app_token" binding:"required"`
	TableId    string   `json:"table_id" binding:"required"`
	ViewId     string   `json:"view_id" binding:"required"`
	FieldNames []string `json:"field_names" binding:"required"` // 需要查询的字段名
	SortOrders string   `json:"sort_orders" binding:"required"` // 根据什么字段排序
	Desc       bool     `json:"desc"`                           // 是否降序
	FilterName string   `json:"filter_name" binding:"required"` // 过滤条件的字段名，根据实际的接口需要，这里只需要设计成一个
	FilterVal  string   `json:"filter_val" binding:"required"`  // 过滤条件的值
	PageToken  string   `json:"pagetoken,omitempty"`            // 分页参数,第一次不需要
}
