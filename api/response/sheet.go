package response

import "github.com/muxi-Infra/FeedBack-Backend/domain"

// GetTableRecordResp 获取表格记录返回参数（个人历史记录）
type GetTableRecordResp struct {
	Records   []domain.TableRecord
	HasMore   bool
	PageToken string
	Total     int
}

// GetFAQProblemTableRecordResp 获取常见问题记录返回参数
type GetFAQProblemTableRecordResp struct {
	Records []domain.FAQTableRecord
	Total   int
}
