package gist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"cfst-client/pkg/models"
)

type Client struct {
	token      string
	prefix     string
	httpClient *http.Client
}

func NewClient(token, proxyPrefix string) *Client {
	return &Client{
		token:  token,
		prefix: proxyPrefix,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) doRequestWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var err error
	var resp *http.Response
	for i := 0; i < maxRetries; i++ {
		resp, err = c.httpClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		status := "unknown"
		if resp != nil {
			status = resp.Status
		}
		log.Printf("[warn] Request to %s failed (attempt %d/%d): err=%v, status=%s", req.URL, i+1, maxRetries, err, status)

		time.Sleep(time.Second * time.Duration(1<<i))
	}
	return resp, fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
}

// [修改] PushResults 函数现在接收 GistContent 对象
func (c *Client) PushResults(gistID, filename string, content models.GistContent) error {
	// [修改] 直接序列化传入的 content 对象
	contentBytes, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal gist content: %w", err)
	}

	body := map[string]interface{}{
		"files": map[string]map[string]string{
			filename: {"content": string(contentBytes)},
		},
	}
	data, _ := json.Marshal(body)

	url := fmt.Sprintf("%shttps://api.github.com/gists/%s", c.prefix, gistID)
	req, _ := http.NewRequest("PATCH", url, bytes.NewReader(data))
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequestWithRetry(req, 3)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("gist patch failed with status: %s", resp.Status)
	}
	return nil
}