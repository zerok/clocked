package backup

type Snapshot struct {
	RawTime  string   `json:"time"`
	Tree     string   `json:"tree"`
	Paths    []string `json:"paths"`
	Hostname string   `json:"hostname"`
	ID       string   `json:"string"`
}
