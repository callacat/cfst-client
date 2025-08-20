package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DeviceName  string `yaml:"device_name"` // [新增]
	ProxyPrefix string `yaml:"proxy_prefix"`

	Gist struct {
		Token  string `yaml:"token"`
		GistID string `yaml:"gist_id"`
	} `yaml:"gist"`

	CF struct {
		Binary     string   `yaml:"binary"`
		Args       []string `yaml:"args"`
		OutputFile string   `yaml:"output_file"`
		Update     struct {
			Check  bool   `yaml:"check"`
			APIURL string `yaml:"api_url"`
		} `yaml:"update"`
	} `yaml:"cf"`
}

// Load 读取并解析配置文件，替换环境变量占位
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	// 环境变量替换
	cfg.ProxyPrefix = os.ExpandEnv(cfg.ProxyPrefix)
	cfg.Gist.Token = os.ExpandEnv(cfg.Gist.Token)
	return &cfg, nil
}