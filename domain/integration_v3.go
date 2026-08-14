package domain

// RegisterProjectInput 是注册项目的业务输入，不与 HTTP 请求结构绑定。
type RegisterProjectInput struct {
	ProjectName string
	School      string
	Tables      []RegisterProjectTableInput
}

type RegisterProjectTableInput struct {
	TableIdentity string
	TableName     string
	TableToken    string
	TableID       string
	ViewID        string
	TableType     string
	Notice        bool
	Scopes        []string
}
