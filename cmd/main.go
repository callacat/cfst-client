// File: cmd/main.go

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync" // [新增] 引入 sync 包
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

// [新增] 创建一个全局互斥锁，用于防止任务并发执行
var runLock sync.Mutex

func main() {
	configPath := filepath.Join(configDir, "config.yml")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("load config:", err)
	}

	if cfg.DeviceName == "" || cfg.LineOperator == "" {
		log.Fatal("Error: 'device_name' and 'line_operator' in config.yml must not be empty.")
	}

	// 立即执行一次测试
	go runAllTests(cfg) // 使用 go 关键字使其在后台运行，避免阻塞后续的定时任务启动

	// 如果配置了 cron 表达式，则启动定时任务
	if cfg.Cron != "" {
		log.Printf("Scheduling tests with cron expression: %s", cfg.Cron)
		c := cron.New()
		_, err := c.AddFunc(cfg.Cron, func() {
			runAllTests(cfg)
		})
		if err != nil {
			log.Fatalf("Error adding cron job: %v", err)
		}
		c.Start()

		// 保持主 goroutine 运行
		select {}
	}
}

func runAllTests(cfg *config.Config) {
	// [核心修改] 尝试获取锁
	// TryLock 是非阻塞的，如果锁已被其他 goroutine 持有，它会立即返回 false
	if !runLock.TryLock() {
		log.Println("A test is already in progress. Skipping this run.")
		return
	}
	// [核心修改] 使用 defer 确保在函数退出时，无论发生什么，都会释放锁
	defer runLock.Unlock()

	log.Println("--- Starting all tests ---")

	var notifiers []notifier.Notifier
	if cfg.Notifications.Enabled {
		if cfg.Notifications.PushPlus.Token != "" {
			notifiers = append(notifiers, &notifier.PushPlusNotifier{Token: cfg.Notifications.PushPlus.Token})
		}
		if cfg.Notifications.Telegram.BotToken != "" && cfg.Notifications.Telegram.ChatID != "" {
			tgNotifier, err := notifier.NewTelegramNotifier(cfg.Notifications.Telegram)
			if err != nil {
				log.Printf("WARN: Failed to initialize Telegram notifier: %v", err)
			} else {
				notifiers = append(notifiers, tgNotifier)
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

	gc := gist.NewClient(os.ExpandEnv(cfg.Gist.Token), cfg.ProxyPrefix)

	log.Println("--- Starting test for IPv4 ---")
	runTest(gc, cfg, "v4", notifiers)

	if cfg.TestIPv6 {
		log.Println("--- Starting test for IPv6 ---")
		runTest(gc, cfg, "v6", notifiers)
	} else {
		log.Println("IPv6 test is disabled in config.yml, skipping.")
	}

	log.Println("--- All tests done ---")
}

// runTest 函数保持不变
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

	finalGistFilename := fmt.Sprintf("%s-%s-%s-%s.json", baseGistFilename, cfg.LineOperator, cfg.DeviceName, version)
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
			log.Printf("Waiting for 5 seconds before next attempt...")
			time.Sleep(5 * time.Second)
		}
	}

	if len(finalResults) == 0 {
		log.Printf("FATAL: Speed test for IP%s failed after %d attempts. No results to upload.", version, cfg.TestOptions.MaxRetries)
		// ... (notification logic remains the same) ...
		return
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
		}
		log.Printf("Gist update for %s failed: %v", finalGistFilename, err)
		return
	}

	log.Printf("--- Test for IP%s completed successfully ---", version)
}