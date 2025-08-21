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

	installer.NewInstaller(cfg.ProxyPrefix, "https://api.github.com/repos/XIU2/CloudflareSpeedTest/releases/latest", cfg.Cf.Binary, configDir).InstallOrUpdate()

	gc := gist.NewClient(os.ExpandEnv(cfg.Gist.Token), cfg.ProxyPrefix)

	log.Println("--- Starting test for IPv4 ---")
	runTest(gc, cfg, "v4")

	if cfg.TestIPv6 {
		log.Println("--- Starting test for IPv6 ---")
		runTest(gc, cfg, "v6")
	} else {
		log.Println("IPv6 test is disabled in config.yml, skipping.")
	}

	log.Println("All tests done.")
}

func runTest(gc *gist.Client, cfg *config.Config, version string) {
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

	finalGistFilename := fmt.Sprintf("%s-%s-%s.json", baseGistFilename, cfg.LineOperator, version)
	finalArgs := append(testConfig.Args, "-f", ipFile)
	localCsvPath := filepath.Join(configDir, testConfig.OutputFile)

	// [CORRECTED] This function call now matches the definition in tester.go
	cf := tester.NewCFSpeedTest(testConfig.Binary, localCsvPath, cfg.DeviceName, finalArgs)
	results, err := cf.Run()
	if err != nil {
		log.Printf("Speed test for IP%s failed: %v", version, err)
		return
	}

	log.Printf("Uploading results to Gist as JSON with filename: %s", finalGistFilename)
	if err := gc.PushResults(cfg.Gist.GistID, finalGistFilename, results); err != nil {
		if strings.Contains(err.Error(), "404") {
			log.Printf("FATAL: Gist update for %s failed with 404 Not Found. Please check Gist ID and GITHUB_TOKEN permissions.", finalGistFilename)
		}
		log.Printf("Gist update for %s failed: %v", finalGistFilename, err)
		return
	}

	log.Printf("--- Test for IP%s completed successfully ---", version)
}