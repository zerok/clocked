package jira

type WorklogCreation struct {
	Started          string `json:"started"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
}
