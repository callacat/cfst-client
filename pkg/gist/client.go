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
	httpClient *http.Client // [新增]
}

func NewClient(token, proxyPrefix string) *Client {
	return &Client{
		token:  token,
		prefix: proxyPrefix,
		// [新增] 初始化一个带超时的 http client
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// [新增] 带重试逻辑的请求函数
func (c *Client) doRequestWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var err error
	var resp *http.Response
	for i := 0; i < maxRetries; i++ {
		resp, err = c.httpClient.Do(req)
		// 如果请求成功且状态码不是服务端错误，则直接返回
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// 打印警告日志
		status := "unknown"
		if resp != nil {
			status = resp.Status
		}
		log.Printf("[warn] Request to %s failed (attempt %d/%d): err=%v, status=%s", req.URL, i+1, maxRetries, err, status)

		// 指数退避等待
		time.Sleep(time.Second * time.Duration(1<<i)) // 1s, 2s, 4s...
	}
	return resp, fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
}

// [修改] PushResults 使用重试逻辑
func (c *Client) PushResults(gistID, filename string, results []models.DeviceResult) error {
	content, _ := json.MarshalIndent(results, "", "  ")
	body := map[string]interface{}{
		"files": map[string]map[string]string{
			filename: {"content": string(content)},
		},
	}
	data, _ := json.Marshal(body)

	url := fmt.Sprintf("%shttps://api.github.com/gists/%s", c.prefix, gistID)
	req, _ := http.NewRequest("PATCH", url, bytes.NewReader(data))
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")

	// 使用重试函数发送请求
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