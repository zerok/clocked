package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
)

const DatetimeFormat = "2006-01-02T15:04:05.000-0700"

type Client struct {
	username string
	password string
	baseURL  string
	log      *logrus.Logger
}

func NewClient(baseURL, username, password string) *Client {
	c := Client{
		username: username,
		password: password,
		baseURL:  baseURL,
		log:      logrus.New(),
	}
	return &c
}

func (c *Client) getSearchResultIssues(ctx context.Context, baseURL string, q url.Values) ([]SearchResultIssue, error) {
	h := http.Client{}
	result := make([]SearchResultIssue, 0, 10)
	var offset int64
	for {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*5)
		q.Set("startAt", fmt.Sprintf("%d", offset))
		u := fmt.Sprintf("%s?%s", baseURL, q.Encode())
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			cancel()
			return nil, err
		}
		c.log.Println(req)
		req.SetBasicAuth(c.username, c.password)
		resp, err := h.Do(req.WithContext(timeoutCtx))
		if err != nil {
			cancel()
			return nil, err
		}
		var searchResult SearchResult
		if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
			resp.Body.Close()
			cancel()
			return nil, err
		}
		resp.Body.Close()
		result = append(result, searchResult.Issues...)
		if searchResult.Total > int64(len(result)) {
			offset += searchResult.MaxResults
		} else {
			cancel()
			break
		}
	}
	return result, nil
}

func (c *Client) getIssueWorklog(ctx context.Context, issueKey string) ([]WorklogResultItem, error) {
	u := fmt.Sprintf("%s/rest/api/2/issue/%s/worklog", c.baseURL, issueKey)
	h := http.Client{}
	// q := url.Values{}
	// var offset int64
	result := make([]WorklogResultItem, 0, 10)
	for {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*5)
		// q.Set("startAt", fmt.Sprintf("%d", offset))
		// req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?%s", u, q.Encode()), nil)
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			cancel()
			return nil, err
		}
		c.log.Println(req)
		req.SetBasicAuth(c.username, c.password)
		resp, err := h.Do(req.WithContext(timeoutCtx))
		if err != nil {
			cancel()
			return nil, err
		}
		var r WorklogResult
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			resp.Body.Close()
			cancel()
			return nil, err
		}
		resp.Body.Close()
		result = append(result, r.Items...)
		break
	}
	return result, nil
}

func (c *Client) RemoveDatedWorklogs(ctx context.Context, date time.Time) error {
	d := date.Format("2006-01-02")
	q := url.Values{}

	q.Set("jql", fmt.Sprintf("worklogAuthor = %s and worklogDate = %s", c.username, d))
	q.Set("fields", "key,id")
	u := fmt.Sprintf("%s/rest/api/2/search", c.baseURL)
	items, err := c.getSearchResultIssues(ctx, u, q)
	if err != nil {
		return err
	}
	tdYear, tdMonth, tdDay := date.Local().Date()
	worklogItems := make([]string, 0, 10)
	for _, item := range items {
		worklog, err := c.getIssueWorklog(context.Background(), item.Key)
		if err != nil {
			return err
		}
		for _, i := range worklog {
			if i.Author.Name != c.username {
				continue
			}
			d, err := time.Parse(DatetimeFormat, i.Started)
			if err != nil {
				return err
			}
			wlYear, wlMonth, wlDay := d.Local().Date()
			if wlYear != tdYear || wlMonth != tdMonth || wlDay != tdDay {
				continue
			}
			worklogItems = append(worklogItems, i.Self)
		}
	}
	for _, i := range worklogItems {
		if err := c.RemoveWorklogItem(ctx, i); err != nil {
			c.log.WithError(err).Errorf("Failed to delete worklog item %s", i)
			return err
		}
	}
	return nil
}

func (c *Client) RemoveWorklogItem(ctx context.Context, itemURL string) error {
	c.log.Infof("Removing worklog item %s", itemURL)
	h := http.Client{}
	req, err := http.NewRequest(http.MethodDelete, itemURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
	resp, err := h.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return nil
	}
	switch resp.StatusCode {
	case 204:
		return nil
	case 403:
		return fmt.Errorf("you are not allowed to delete this entry")
	case 400:
		return fmt.Errorf("invalid input")
	default:
		return fmt.Errorf("unexpected return code %d", resp.StatusCode)
	}
}

func (c *Client) AddWorklog(ctx context.Context, taskID string, start time.Time, dur time.Duration) error {
	u := fmt.Sprintf("%s/rest/api/2/issue/%s/worklog", c.baseURL, taskID)
	h := http.Client{}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WorklogCreation{
		Started:          start.UTC().Format(DatetimeFormat),
		TimeSpentSeconds: int64(dur.Round(time.Second).Seconds()),
		Comment:          fmt.Sprintf("Working on %s", taskID),
	}); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, u, &body)
	if err != nil {
		return err
	}
	c.log.Info(req)
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
