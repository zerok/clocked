package jira

type SearchResultIssue struct {
	Key string `json:"key"`
	ID  string `json:"id"`
}

type SearchResult struct {
	Total      int64               `json:"total"`
	StartAt    int64               `json:"startAt"`
	MaxResults int64               `json:"maxResults"`
	Issues     []SearchResultIssue `json:"issues"`
}
