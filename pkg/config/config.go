// File: pkg/config/config.go

package config

import (
	"gopkg.in/yaml.v2"
	"os"
)

// CfConfig 定义了 CloudflareSpeedTest 的相关配置
type CfConfig struct {
	Binary     string   `yaml:"binary"`
	Args       []string `yaml:"args"`
	OutputFile string   `yaml:"output_file"`
}

// UpdateConfig 定义了自动更新的配置
type UpdateConfig struct {
	Check  bool   `yaml:"check"`
	ApiURL string `yaml:"api_url"`
}

// Config 是整个应用的配置结构
type Config struct {
	DeviceName   string `yaml:"device_name"`
	LineOperator string `yaml:"line_operator"`
	TestIPv6     bool   `yaml:"test_ipv6"`
	ProxyPrefix  string `yaml:"proxy_prefix"`

	Gist struct {
		Token  string `yaml:"token"`
		GistID string `yaml:"gist_id"`
	} `yaml:"gist"`

	TestOptions struct {
        MinResults      int `yaml:"min_results"`
        MaxRetries      int `yaml:"max_retries"`
        GistUploadLimit int `yaml:"gist_upload_limit"`
    } `yaml:"test_options"`

    Cf     CfConfig     `yaml:"cf"`
    Cf6    CfConfig     `yaml:"cf6"`
    Update UpdateConfig `yaml:"update"`
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
	cfg.ProxyPrefix = os.ExpandEnv(cfg.ProxyPrefix)
	cfg.Gist.Token = os.ExpandEnv(cfg.Gist.Token)
	return &cfg, nil
}

type Config struct {
    // ...
    Notifications struct {
        Enabled  bool `yaml:"enabled"`
        PushPlus struct {
            Token string `yaml:"token"`
        } `yaml:"pushplus"`
        Telegram struct {
            BotToken string `yaml:"bot_token"`
            ChatID   string `yaml:"chat_id"`
        } `yaml:"telegram"`
    } `yaml:"notifications"`
}