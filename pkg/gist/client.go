package gist

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "cfst-client/pkg/models"
)

type Client struct {
    token  string
    prefix string
}

func NewClient(token, proxyPrefix string) *Client {
    return &Client{token: token, prefix: proxyPrefix}
}

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

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return fmt.Errorf("gist patch failed: %s", resp.Status)
    }
    return nil
}
