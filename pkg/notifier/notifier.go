// File: pkg/notifier/notifier.go
package notifier

import "log"

// Notifier 定义了通知器的通用接口
type Notifier interface {
	Notify(title, message string) error
}

// PushPlusNotifier 实现了 PushPlus 通知
type PushPlusNotifier struct {
	Token string
}

func (p *PushPlusNotifier) Notify(title, message string) error {
	// 在这里实现调用 PushPlus API 的逻辑
	log.Printf("PushPlus notification sent (mock): Title=%s, Message=%s", title, message)
	return nil
}

// TelegramNotifier 实现了 Telegram Bot 通知
type TelegramNotifier struct {
	BotToken string
	ChatID   string
}

func (t *TelegramNotifier) Notify(title, message string) error {
	// 在这里实现调用 Telegram Bot API 的逻辑
	log.Printf("Telegram notification sent (mock): Title=%s, Message=%s", title, message)
	return nil
}