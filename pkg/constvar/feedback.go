package constvar

const (
	// Gin Context 中保存 V3 用户身份声明的键。
	V3ClaimsContextKey = "v3_claims"
	// Gin Context 中保存 V3 管理员 ID 的键。
	AdminIDContextKeyV3 = "admin_id_v3"

	// 反馈项目表类型。
	FeedbackTableType = "feedback"
	FAQTableType      = "faq"

	// 项目表 Scope。
	FeedbackScopeCreate   = "feedback:create"
	FeedbackScopeReadSelf = "feedback:read:self"
	FeedbackScopeRead     = "feedback:read"
	FeedbackScopeWrite    = "feedback:write"
	FeedbackScopeSync     = "feedback:sync"
)

// SupportedFeedbackScopes 返回当前支持的反馈权限集合。
func SupportedFeedbackScopes() map[string]struct{} {
	return map[string]struct{}{
		FeedbackScopeCreate:   {},
		FeedbackScopeReadSelf: {},
		FeedbackScopeRead:     {},
		FeedbackScopeWrite:    {},
		FeedbackScopeSync:     {},
	}
}
