package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cfst-client/pkg/config"
	"cfst-client/pkg/gist"
	"cfst-client/pkg/installer"
	"cfst-client/pkg/tester"
)

const configDir = "/app/config"

func main() {
	configPath := filepath.Join(configDir, "config.yml")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("load config:", err)
	}

	if cfg.DeviceName == "" || cfg.LineOperator == "" {
		log.Fatal("Error: 'device_name' and 'line_operator' in config.yml must not be empty.")
	}

	// 自动更新逻辑只需执行一次
	installer.NewInstaller(cfg.ProxyPrefix, "https://api.github.com/repos/XIU2/CloudflareSpeedTest/releases/latest", cfg.Cf.Binary, configDir).InstallOrUpdate()

	gc := gist.NewClient(os.ExpandEnv(cfg.Gist.Token), cfg.ProxyPrefix)

	// --- 1. 执行 IPv4 测试 (默认) ---
	log.Println("--- Starting test for IPv4 ---")
	runTest(gc, cfg, "v4")

	// --- 2. 检查并执行 IPv6 测试 ---
	if cfg.TestIPv6 {
		log.Println("--- Starting test for IPv6 ---")
		runTest(gc, cfg, "v6")
	} else {
		log.Println("IPv6 test is disabled in config.yml, skipping.")
	}

	log.Println("All tests done.")
}

// [新增] runTest 函数封装了完整的 “测速 -> 上传” 流程
// version 参数可以是 "v4" 或 "v6"
func runTest(gc *gist.Client, cfg *config.Config, version string) {
	var testConfig config.CfConfig
	var ipFile string
	var baseGistFilename string

	// 根据版本选择不同的配置和文件
	if version == "v6" {
		testConfig = cfg.Cf6
		ipFile = filepath.Join(configDir, "ipv6.txt")
		baseGistFilename = "results6" // 上传到 Gist 的基础文件名
	} else {
		testConfig = cfg.Cf
		ipFile = filepath.Join(configDir, "ip.txt")
		baseGistFilename = "results" // 上传到 Gist 的基础文件名
	}

	// 1. 动态生成最终要上传到 Gist 的 JSON 文件名
	// 例如: "results-cu-v4.json" 或 "results6-cu-v6.json"
	finalGistFilename := fmt.Sprintf("%s-%s-%s.json", baseGistFilename, cfg.LineOperator, version)

	// 2. 将 -f 参数和对应的 IP 文件附加到参数列表
	finalArgs := append(testConfig.Args, "-f", ipFile)
	
	// 3. 执行测速，使用配置中指定的本地 CSV 输出文件名
	localCsvPath := filepath.Join(configDir, testConfig.OutputFile)
	cf := tester.NewCFSpeedTest(testConfig.Binary, localCsvPath, cfg.DeviceName, finalArgs)
	results, err := cf.Run()
	if err != nil {
		log.Printf("Speed test for IP%s failed: %v", version, err)
		return // 如果测试失败，直接返回
	}
	
	// 4. 上传结果到 Gist
	log.Printf("Uploading results to Gist as JSON with filename: %s", finalGistFilename)
	if err := gc.PushResults(cfg.Gist.GistID, finalGistFilename, results); err != nil {
		if strings.Contains(err.Error(), "404") {
			log.Printf("FATAL: Gist update for %s failed with 404 Not Found. Please check Gist ID and GITHUB_TOKEN permissions.", finalGistFilename)
		}
		// 使用 log.Printf 而不是 log.Fatalf，以避免 v4 失败时 v6 无法执行
		log.Printf("Gist update for %s failed: %v", finalGistFilename, err)
		return
	}
	
	log.Printf("--- Test for IP%s completed successfully ---", version)
}