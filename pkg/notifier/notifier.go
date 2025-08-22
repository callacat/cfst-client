// File: pkg/notifier/notifier.go
package notifier

import (
	"bytes"
	"cfst-client/pkg/config"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// Notifier 定义了通知器的通用接口
type Notifier interface {
	Notify(title, message string) error
}

// TelegramNotifier 实现了 Telegram Bot 通知
type TelegramNotifier struct {
	BotToken   string
	ChatID     string
	apiURL     string
	httpClient *http.Client
}

// NewTelegramNotifier 创建一个新的 Telegram 通知器实例
func NewTelegramNotifier(cfg config.TelegramConfig) (*TelegramNotifier, error) {
	// 默认 API 地址
	apiBaseURL := "https://api.telegram.org"

	// 创建 http client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// [核心] 根据配置设置代理
	if cfg.Proxy.Enabled {
		switch cfg.Proxy.Type {
		case "socks5":
			if cfg.Proxy.Address == "" {
				return nil, fmt.Errorf("socks5 proxy address is not set")
			}
			log.Println("Telegram Notifier: Using SOCKS5 proxy:", cfg.Proxy.Address)
			dialer, err := proxy.SOCKS5("tcp", cfg.Proxy.Address, nil, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("failed to create socks5 dialer: %w", err)
			}
			transport := &http.Transport{
				Dial: dialer.Dial,
			}
			client.Transport = transport
		case "reverse_proxy":
			if cfg.Proxy.ApiURL == "" {
				return nil, fmt.Errorf("reverse proxy api_url is not set")
			}
			log.Println("Telegram Notifier: Using reverse proxy:", cfg.Proxy.ApiURL)
			// 如果反代链接末尾有斜杠，则去掉
			apiBaseURL = cfg.Proxy.ApiURL
			if apiBaseURL[len(apiBaseURL)-1] == '/' {
				apiBaseURL = apiBaseURL[:len(apiBaseURL)-1]
			}
		default:
			return nil, fmt.Errorf("invalid telegram proxy type: %s", cfg.Proxy.Type)
		}
	}

	return &TelegramNotifier{
		BotToken:   cfg.BotToken,
		ChatID:     cfg.ChatID,
		apiURL:     fmt.Sprintf("%s/bot%s/sendMessage", apiBaseURL, cfg.BotToken),
		httpClient: client,
	}, nil
}

// Notify 发送通知
func (t *TelegramNotifier) Notify(title, message string) error {
	fullMessage := fmt.Sprintf("<b>%s</b>\n\n%s", title, message)

	body, _ := json.Marshal(map[string]string{
		"chat_id":    t.ChatID,
		"text":       fullMessage,
		"parse_mode": "HTML",
	})

	req, err := http.NewRequest("POST", t.apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram notification failed with status: %s", resp.Status)
	}

	log.Println("Telegram notification sent successfully.")
	return nil
}

// Mock PushPlus Notifier for completeness
type PushPlusNotifier struct {
	Token string
}

func (p *PushPlusNotifier) Notify(title, message string) error {
	log.Printf("PushPlus notification sent (mock): Title=%s", title)
	return nil
}