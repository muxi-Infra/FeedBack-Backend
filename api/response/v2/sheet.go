package v2

import "github.com/muxi-Infra/FeedBack-Backend/domain"

// GetTableRecordResp 获取表格记录返回参数（个人历史记录）
type GetTableRecordResp struct {
	Records   []domain.TableRecord `json:"records"`
	HasMore   bool                 `json:"has_more"`
	PageToken string               `json:"page_token"`
}
