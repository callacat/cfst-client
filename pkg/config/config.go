package config

import (
	"os"
	"gopg.in/yaml.v2"
)

// CfConfig 定义了 CloudflareSpeedTest 的相关配置
type CfConfig struct {
	Binary     string   `yaml:"binary"`
	Args       []string `yaml:"args"`
	OutputFile string   `yaml:"output_file"`
}

// Config 是整个应用的配置结构
type Config struct {
	DeviceName   string `yaml:"device_name"`
	LineOperator string `yaml:"line_operator"`
	TestIPv6     bool   `yaml:"test_ipv6"` // [新增]
	ProxyPrefix  string `yaml:"proxy_prefix"`

	Gist struct {
		Token  string `yaml:"token"`
		GistID string `yaml:"gist_id"`
	} `yaml:"gist"`

	Cf  CfConfig `yaml:"cf"`  // [修改] IPv4 配置
	Cf6 CfConfig `yaml:"cf6"` // [新增] IPv6 配置
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
	cfg.ProxyPrefix = os.ExpandEnv(cfg.ProxyPrefix)
	cfg.Gist.Token = os.ExpandEnv(cfg.Gist.Token)
	return &cfg, nil
}