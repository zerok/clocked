package jira

type WorklogCreation struct {
	Started          string `json:"started"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
	Comment          string `json:"comment"`
}

type WorklogResultItem struct {
	Author struct {
		Name string `json:"name"`
	} `json:"author"`
	ID      string `json:"id"`
	Started string `json:"started"`
	Self    string `json:"self"`
}

type WorklogResult struct {
	MaxResults int64               `json:"maxResults"`
	Total      int64               `json:"total"`
	StartAt    int64               `json:"startAt"`
	Items      []WorklogResultItem `json:"worklogs"`
}
