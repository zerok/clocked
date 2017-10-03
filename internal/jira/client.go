package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	username string
	password string
	baseURL  string
}

func NewClient(baseURL, username, password string) *Client {
	c := Client{
		username: username,
		password: password,
		baseURL:  baseURL,
	}
	return &c
}

func (c *Client) AddWorklog(ctx context.Context, taskID string, start time.Time, dur time.Duration) error {
	u := fmt.Sprintf("%s/rest/api/2/issue/%s/worklog", c.baseURL, taskID)
	fmt.Println(u)
	h := http.Client{}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WorklogCreation{
		Started:          start.Format(time.RFC3339),
		TimeSpentSeconds: int64(dur.Seconds()),
		Comment:          fmt.Sprintf("Working on %s", taskID),
	}); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, u, &body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-type", "application/json")
	resp, err := h.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		err := fmt.Errorf("failed to create worklog: status code %v returned", resp.StatusCode)
		io.Copy(os.Stdout, resp.Body)
		return err
	}
	return nil
}
