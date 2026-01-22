package domain

type LarkMessage struct {
	Type string          `json:"type"`
	Data LarkMessageData `json:"data"`
}

type LarkMessageData struct {
	TemplateId          string                 `json:"template_id"`
	TemplateVersionName string                 `json:"template_version_name"`
	TemplateVariable    map[string]interface{} `json:"template_variable"`
}

type CCNUBoxFeedMessage struct {
	Content   string `json:"content"`
	StudentID string `json:"student_id"`
	Title     string `json:"title"`
	Type      string `json:"type"`
}
