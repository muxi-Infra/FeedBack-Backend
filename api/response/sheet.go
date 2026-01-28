package response

import "github.com/muxi-Infra/FeedBack-Backend/domain"

// CreatTableRecordResp 创建表格记录返回参数
type CreatTableRecordResp struct {
	RecordID string `json:"record_id"`
}

// GetTableRecordResp 获取表格记录返回参数（个人历史记录）
type GetTableRecordResp struct {
	Records   []domain.TableRecord `json:"records"`
	HasMore   bool                 `json:"has_more"`
	PageToken string               `json:"page_token"`
	Total     int                  `json:"total"`
}

// GetFAQProblemTableRecordResp 获取常见问题记录返回参数
type GetFAQProblemTableRecordResp struct {
	Records []domain.FAQTableRecord `json:"records"`
	Total   int                     `json:"total"`
}

// GetPhotoUrlResp 获取图片URL返回参数
type GetPhotoUrlResp struct {
	Files []domain.File `json:"files"`
}
