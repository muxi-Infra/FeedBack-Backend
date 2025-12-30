package request

type CreateAppTableRecordReq struct {
	// 修改为配置，而不是请求体中字段了
	//AppToken string `json:"app_token" binding:"required"`
	//TableId  string `json:"table_id" binding:"required"`
	//UserIdType             string                 `json:"user_id_type,omitempty"`
	//ClientToken            string                 `json:"client_token,omitempty"`

	IgnoreConsistencyCheck bool                   `json:"ignore_consistency_check,omitempty"`
	Fields                 map[string]interface{} `json:"fields"` // 记录的字段 不再校验，required,由后端自动填充

	// 必填字段
	StudentID string `json:"student_id" binding:"required" feishu:"用户ID"` // 即学号
	Contact   string `json:"contact" binding:"required" feishu:"联系方式（QQ/邮箱）"`
	Content   string `json:"content" binding:"required" feishu:"反馈内容"`

	// 可选字段
	ScreenShot    []ScreenShot `json:"screen_shot,omitempty" feishu:"截图"`
	ProblemType   string       `json:"problem_type,omitempty" feishu:"问题类型"`
	ProblemSource string       `json:"problem_source,omitempty" feishu:"问题来源"`

	// 自动补充
	SubmitTIme int64  `json:"-" feishu:"提交时间"` // 提交时间
	Status     string `json:"-" feishu:"问题状态"` // "处理中“
}

// ScreenShot 附件上传是需要对象的形式
type ScreenShot struct {
	FileToken string `json:"file_token"`
}

type GetAppTableRecordReq struct {
	// 同理 从配置中获取
	//AppToken   string   `json:"app_token" binding:"required"`
	//TableId    string   `json:"table_id" binding:"required"`
	//ViewId     string   `json:"view_id" binding:"required"`

	StudentID  string   `json:"student_id"`                     // 即学号 // 用于查看相应学号的人是否点赞某条记录了没有
	FieldNames []string `json:"field_names" binding:"required"` // 需要查询的字段名
	SortOrders string   `json:"sort_orders" binding:"required"` // 根据什么字段排序
	Desc       bool     `json:"desc"`                           // 是否降序
	FilterName string   `json:"filter_name"`                    // 过滤条件的字段名，根据实际的接口需要，这里只需要设计成一个
	FilterVal  string   `json:"filter_val"`                     // 过滤条件的值
	PageToken  string   `json:"pagetoken,omitempty"`            // 分页参数,第一次不需要
}

type GetPhotoUrlReq struct {
	FileTokens []string `json:"file_tokens" binding:"required"` // 附件token
}
