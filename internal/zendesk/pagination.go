package zendesk

type Meta struct {
	HasMore      bool   `json:"has_more"`
	AfterCursor  string `json:"after_cursor"`
	BeforeCursor string `json:"before_cursor"`
}
type Links struct {
	Prev string `json:"prev"`
	Next string `json:"next"`
}
