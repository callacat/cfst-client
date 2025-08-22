// File: cmd/main.go

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
	"cfst-client/pkg/models"
	"cfst-client/pkg/notifier"
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

	log.Println("All tests done.")
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

	finalGistFilename := fmt.Sprintf("%s-%s-%s-%s.json", baseGistFilename, cfg.LineOperator, cfg.DeviceName, version)
	finalArgs := append(testConfig.Args, "-f", ipFile)
	localCsvPath := filepath.Join(configDir, testConfig.OutputFile)

	// [FIX] Pass cfg.LineOperator as the fourth argument to the speed tester
	cf := tester.NewCFSpeedTester(testConfig.Binary, localCsvPath, cfg.DeviceName, cfg.LineOperator, finalArgs)

	var results []models.DeviceResult
	var err error

	for i := 0; i < cfg.TestOptions.MaxRetries; i++ {
		log.Printf("--- Starting speed test for IP%s (Attempt %d/%d) ---", version, i+1, cfg.TestOptions.MaxRetries)
		results, err = cf.Run()
		if err != nil {
			log.Printf("Speed test for IP%s failed on attempt %d: %v", version, i+1, err)
			continue
		}

		if len(results) >= cfg.TestOptions.MinResults {
			log.Printf("Successfully got %d results, which meets the minimum requirement of %d.", len(results), cfg.TestOptions.MinResults)
			break
		}

		log.Printf("WARN: Got only %d results, which is less than the required minimum of %d. Retrying...", len(results), cfg.TestOptions.MinResults)
		results = nil
	}

	if len(results) == 0 {
		log.Printf("FATAL: Speed test for IP%s failed after %d attempts. No results to upload.", version, cfg.TestOptions.MaxRetries)
		title := fmt.Sprintf("Speed Test Failed on %s", cfg.DeviceName)
		message := fmt.Sprintf("The %s speed test for IP%s failed after %d attempts.", cfg.LineOperator, version, cfg.TestOptions.MaxRetries)
		for _, n := range notifiers {
			if err := n.Notify(title, message); err != nil {
				log.Printf("Failed to send notification: %v", err)
			}
		}
		return
	}

	var uploadResults []models.DeviceResult
	if len(results) > cfg.TestOptions.GistUploadLimit {
		log.Printf("Result count (%d) exceeds the limit (%d). Truncating to the top %d.", len(results), cfg.TestOptions.GistUploadLimit, cfg.TestOptions.GistUploadLimit)
		uploadResults = results[:cfg.TestOptions.GistUploadLimit]
	} else {
		uploadResults = results
	}

	log.Printf("Uploading %d results to Gist as JSON with filename: %s", len(uploadResults), finalGistFilename)
	if err := gc.PushResults(cfg.Gist.GistID, finalGistFilename, uploadResults); err != nil {
		if strings.Contains(err.Error(), "404") {
			log.Printf("FATAL: Gist update for %s failed with 404 Not Found. Please check Gist ID and GITHUB_TOKEN permissions.", finalGistFilename)
		}
		log.Printf("Gist update for %s failed: %v", finalGistFilename, err)
		return
	}

	log.Printf("--- Test for IP%s completed successfully ---", version)
}