package constvar

// 管理后台角色。
const (
	AdminRoleSuperAdmin   = "super_admin"
	AdminRoleProjectAdmin = "project_admin"
	AdminRoleOperator     = "operator"
	AdminRoleViewer       = "viewer"
)

// 管理后台权限动作。
const (
	AdminActionRead   = "read"
	AdminActionCreate = "create"
	AdminActionUpdate = "update"
	AdminActionDelete = "delete"
)

// 管理后台资源。
const (
	AdminObjectProject = "project"
	AdminObjectAdmin   = "admin"
	AdminObjectKey     = "key"
	AdminObjectTable   = "table"
	AdminObjectScope   = "scope"
	AdminObjectAudit   = "audit"
)

// 管理员账号状态。
const (
	AdminStatusActive   = "active"
	AdminStatusLocked   = "locked"
	AdminStatusDisabled = "disabled"
)
