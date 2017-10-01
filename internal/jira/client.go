package jira

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	u := fmt.Sprintf("%s/rest/tempo-rest/1.0/worklogs/%s", c.baseURL, taskID)
	h := http.Client{}
	form := url.Values{}
	form.Set("ansidate", start.Format("2006-01-02"))
	form.Set("ansienddate", start.Add(dur).Format("2006-01-02"))
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
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
