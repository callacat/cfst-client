// File: pkg/config/config.go

package config

import (
	"gopkg.in/yaml.v2"
	"os"
)

// ... (其他结构体不变) ...

type CfConfig struct {
	Binary     string   `yaml:"binary"`
	Args       []string `yaml:"args"`
	OutputFile string   `yaml:"output_file"`
}

type UpdateConfig struct {
	Check  bool   `yaml:"check"`
	ApiURL string `yaml:"api_url"`
}

// [新增] 延迟重试的配置结构体
type DelayedRetryConfig struct {
	Enabled      bool `yaml:"enabled"`
	DelayMinutes int  `yaml:"delay_minutes"`
}

type TestOptions struct {
	MinResults      int `yaml:"min_results"`
	MaxRetries      int `yaml:"max_retries"`
	GistUploadLimit int `yaml:"gist_upload_limit"`
	RetryDelay      int `yaml:"retry_delay"` 
	// [新增] 嵌入延迟重试的配置
	DelayedRetry    DelayedRetryConfig `yaml:"delayed_retry"`
}

// ... (其他结构体不变) ...
type TelegramProxyConfig struct {
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type"`
	Address string `yaml:"address"`
	ApiURL  string `yaml:"api_url"`
}

type TelegramConfig struct {
	BotToken string              `yaml:"bot_token"`
	ChatID   string              `yaml:"chat_id"`
	Proxy    TelegramProxyConfig `yaml:"proxy"`
}

type NotificationsConfig struct {
	Enabled  bool           `yaml:"enabled"`
	PushPlus struct {
		Token string `yaml:"token"`
	} `yaml:"pushplus"`
	Telegram TelegramConfig `yaml:"telegram"`
}

// Config 是整个应用的配置结构
type Config struct {
	DeviceName   string `yaml:"device_name"`
	LineOperator string `yaml:"line_operator"`
	TestIPv6     bool   `yaml:"test_ipv6"`
	ProxyPrefix  string `yaml:"proxy_prefix"`
	Cron         string `yaml:"cron"`

	Gist struct {
		Token  string `yaml:"token"`
		GistID string `yaml:"gist_id"`
	} `yaml:"gist"`

	Notifications NotificationsConfig `yaml:"notifications"`
	TestOptions   TestOptions         `yaml:"test_options"`
	Cf            CfConfig            `yaml:"cf"`
	Cf6           CfConfig            `yaml:"cf6"`
	Update        UpdateConfig        `yaml:"update"`
}

// Load 读取并解析配置文件
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	// 展开所有需要使用环境变量的字段
	cfg.ProxyPrefix = os.ExpandEnv(cfg.ProxyPrefix)
	cfg.Gist.Token = os.ExpandEnv(cfg.Gist.Token)
	cfg.Notifications.Telegram.BotToken = os.ExpandEnv(cfg.Notifications.Telegram.BotToken)
	cfg.Notifications.Telegram.ChatID = os.ExpandEnv(cfg.Notifications.Telegram.ChatID)

	if cfg.TestOptions.RetryDelay <= 0 {
		cfg.TestOptions.RetryDelay = 5 // 默认为 5 秒
	}

	return &cfg, nil
}
