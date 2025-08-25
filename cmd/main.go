// File: cmd/main.go

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cfst-client/pkg/config"
	"cfst-client/pkg/gist"
	"cfst-client/pkg/installer"
	"cfst-client/pkg/models"
	"cfst-client/pkg/notifier"
	"cfst-client/pkg/tester"
	"github.com/robfig/cron/v3"
)

const configDir = "/app/config"

var (
	runLock    sync.Mutex
	configPath = filepath.Join(configDir, "config.yml")
)

// [新增] 全局变量，以便延迟任务可以访问它们
var (
	globalGistClient *gist.Client
	globalNotifiers  []notifier.Notifier
)

func main() {
	// 立即执行一次测试
	go runAllTests()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load initial config: %v. Please check the config file.", err)
	}

	if cfg.Cron != "" {
		log.Printf("Scheduling tests with cron expression: %s", cfg.Cron)
		c := cron.New()
		_, err := c.AddFunc(cfg.Cron, runAllTests)
		if err != nil {
			log.Fatalf("Error adding cron job: %v", err)
		}
		c.Start()
		select {}
	}
}

func runAllTests() {
	if !runLock.TryLock() {
		log.Println("A test is already in progress. Skipping this run.")
		return
	}
	defer runLock.Unlock()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("ERROR: Failed to reload config: %v. Skipping this run.", err)
		return
	}

	log.Println("--- Starting all tests with latest configuration ---")

	if cfg.DeviceName == "" || cfg.LineOperator == "" {
		log.Println("ERROR: 'device_name' and 'line_operator' in config.yml must not be empty. Skipping this run.")
		return
	}

	// [修改] 初始化全局通知器列表
	globalNotifiers = nil
	if cfg.Notifications.Enabled {
		if cfg.Notifications.PushPlus.Token != "" {
			globalNotifiers = append(globalNotifiers, &notifier.PushPlusNotifier{Token: cfg.Notifications.PushPlus.Token})
		}
		if cfg.Notifications.Telegram.BotToken != "" && cfg.Notifications.Telegram.ChatID != "" {
			tgNotifier, err := notifier.NewTelegramNotifier(cfg.Notifications.Telegram)
			if err != nil {
				log.Printf("WARN: Failed to initialize Telegram notifier: %v", err)
			} else {
				globalNotifiers = append(globalNotifiers, tgNotifier)
			}
		}
	}

	if cfg.Update.Check {
		log.Println("--- Checking for CloudflareSpeedTest updates ---")
		err := installer.NewInstaller(cfg.ProxyPrefix, cfg.Update.ApiURL, cfg.Cf.Binary, configDir).InstallOrUpdate()
		if err != nil {
			log.Printf("WARN: Failed to update CloudflareSpeedTest: %v", err)
		} else {
			log.Println("--- Update check finished ---")
		}
	} else {
		log.Println("CloudflareSpeedTest update check is disabled in config.yml.")
	}

	// [修改] 初始化全局 Gist 客户端
	globalGistClient = gist.NewClient(os.ExpandEnv(cfg.Gist.Token), cfg.ProxyPrefix)

	log.Println("--- Starting test for IPv4 ---")
	runTest(globalGistClient, cfg, "v4", globalNotifiers)

	if cfg.TestIPv6 {
		log.Println("--- Starting test for IPv6 ---")
		runTest(globalGistClient, cfg, "v6", globalNotifiers)
	} else {
		log.Println("IPv6 test is disabled in config.yml, skipping.")
	}

	log.Println("--- All tests done ---")
}

// [新增] 用于执行延迟重试的函数
func scheduleDelayedRetry(version string) {
	// 重新加载最新的配置，以防用户在等待期间修改了配置
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("DELAYED RETRY [IP%s]: ERROR: Failed to reload config: %v. Aborting delayed retry.", version, err)
		return
	}

	delay := time.Duration(cfg.TestOptions.DelayedRetry.DelayMinutes) * time.Minute
	log.Printf("DELAYED RETRY [IP%s]: Test failed. Scheduling a delayed retry in %v.", version, delay)

	time.AfterFunc(delay, func() {
		log.Printf("DELAYED RETRY [IP%s]: Starting delayed retry now.", version)
		if !runLock.TryLock() {
			log.Printf("DELAYED RETRY [IP%s]: Another test is already in progress. Skipping delayed retry.", version)
			return
		}
		defer runLock.Unlock()

		// 使用最新的配置和全局客户端/通知器执行单次测试
		runTest(globalGistClient, cfg, version, globalNotifiers)
	})
}

func runTest(gc *gist.Client, cfg *config.Config, version string, notifiers []notifier.Notifier) {
	var testConfig config.CfConfig
	var ipFile string
	var baseGistFilename string

	if version == "v6" {
		testConfig = cfg.Cf6
		ipFile = filepath.Join(configDir, "ipv6.txt")
		baseGistFilename = "results6"
	} else {
		testConfig = cfg.Cf
		ipFile = filepath.Join(configDir, "ip.txt")
		baseGistFilename = "results"
	}

	finalGistFilename := fmt.Sprintf("%s-%s.json", baseGistFilename, cfg.LineOperator)
	finalArgs := append(testConfig.Args, "-f", ipFile)
	localCsvPath := filepath.Join(configDir, testConfig.OutputFile)

	cf := tester.NewCFSpeedTester(testConfig.Binary, localCsvPath, cfg.DeviceName, cfg.LineOperator, finalArgs)

	var finalResults []models.DeviceResult
	for i := 0; i < cfg.TestOptions.MaxRetries; i++ {
		log.Printf("--- Starting speed test for IP%s (Attempt %d/%d) ---", version, i+1, cfg.TestOptions.MaxRetries)
		currentResults, err := cf.Run()

		if err != nil {
			log.Printf("Speed test for IP%s failed on attempt %d: %v", version, i+1, err)
		} else if len(currentResults) > 0 {
			log.Printf("Got %d results in this attempt.", len(currentResults))
			finalResults = currentResults
		}

		if len(finalResults) >= cfg.TestOptions.MinResults {
			log.Printf("Got enough results (%d). Proceeding to upload.", len(finalResults))
			break
		}

		if i < cfg.TestOptions.MaxRetries-1 {
			delay := time.Duration(cfg.TestOptions.RetryDelay) * time.Second
			log.Printf("Waiting for %v before next attempt...", delay)
			time.Sleep(delay)
		}
	}

	if len(finalResults) == 0 {
		log.Printf("FATAL: Speed test for IP%s failed after %d immediate attempts.", version, cfg.TestOptions.MaxRetries)
		// [新增] 检查是否启用延迟重试
		if cfg.TestOptions.DelayedRetry.Enabled && cfg.TestOptions.DelayedRetry.DelayMinutes > 0 {
			// 在一个新的 goroutine 中安排延迟重试，不会阻塞后续代码
			go scheduleDelayedRetry(version)
		}
		return // 结束当前测试流程
	}

	log.Println("Sorting final results...")
	sort.Slice(finalResults, func(i, j int) bool {
		if finalResults[i].LossPct != finalResults[j].LossPct {
			return finalResults[i].LossPct < finalResults[j].LossPct
		}
		if finalResults[i].LatencyMs != finalResults[j].LatencyMs {
			return finalResults[i].LatencyMs < finalResults[j].LatencyMs
		}
		return finalResults[i].DLMBps > finalResults[j].DLMBps
	})

	var uploadResults []models.DeviceResult
	if len(finalResults) > cfg.TestOptions.GistUploadLimit {
		log.Printf("Total result count (%d) exceeds the limit (%d). Truncating to the top %d best results.", len(finalResults), cfg.TestOptions.GistUploadLimit, cfg.TestOptions.GistUploadLimit)
		uploadResults = finalResults[:cfg.TestOptions.GistUploadLimit]
	} else {
		uploadResults = finalResults
	}

	gistContent := models.GistContent{
		Timestamp: time.Now().Format(time.RFC3339),
		Results:   uploadResults,
	}

	log.Printf("Uploading %d results to Gist as JSON with filename: %s", len(uploadResults), finalGistFilename)
	if err := gc.PushResults(cfg.Gist.GistID, finalGistFilename, gistContent); err != nil {
		if strings.Contains(err.Error(), "404") {
			log.Printf("FATAL: Gist update for %s failed with 404 Not Found. Please check Gist ID and GITHUB_TOKEN permissions.", finalGistFilename)
		} else {
			log.Printf("Gist update for %s failed: %v", finalGistFilename, err)
		}
		return
	}

	log.Printf("--- Test for IP%s completed successfully ---", version)
}
