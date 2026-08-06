package domain

import "time"

// CreateAdminInput 创建管理员账号所需的信息。
// Password 是明文输入，只允许在 Service 层短暂使用，不能进入 Repository 或日志。
type CreateAdminInput struct {
	Username    string
	DisplayName string
	Password    string
}

// AdminLoginInput 管理员登录信息。
type AdminLoginInput struct {
	Username string
	Password string
}

// ChangeAdminPasswordInput 修改管理员密码所需的信息。
type ChangeAdminPasswordInput struct {
	AdminID         uint64
	CurrentPassword string
	NewPassword     string
}

// AdminUser 管理员账号的对外信息，不包含密码哈希。
type AdminUser struct {
	ID          uint64
	Username    string
	DisplayName string
	Status      string
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AdminLoginResult 管理员登录结果。
type AdminLoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      AdminUser
}

// AdminTokenClaims 是 Admin JWT 解析后的身份信息。
type AdminTokenClaims struct {
	AdminID   uint64
	ExpiresAt time.Time
}
