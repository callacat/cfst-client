package main

import (
	"log"
	"os"
	"path/filepath"

	"cfst-client/pkg/config"
	"cfst-client/pkg/gist"
	"cfst-client/pkg/installer"
	"cfst-client/pkg/tester"
)

// [修改] 使用容器内的绝对路径作为基准
const configDir = "/app/config"

func main() {
	configPath := filepath.Join(configDir, "config.yml")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("load config:", err)
	}

	if cfg.CF.Update.Check {
		inst := installer.NewInstaller(
			cfg.ProxyPrefix,
			cfg.CF.Update.APIURL,
			cfg.CF.Binary,
			configDir,
		)
		if err := inst.InstallOrUpdate(); err != nil {
			log.Fatalf("install/update cf failed: %v", err)
		}
	}

	outputFilePath := filepath.Join(configDir, cfg.CF.OutputFile)
	cf := tester.NewCFSpeedTester(cfg.CF.Binary, outputFilePath, cfg.DeviceName, cfg.CF.Args)
	results, err := cf.Run()
	if err != nil {
		log.Fatal("speed test failed:", err)
	}

	gc := gist.NewClient(os.ExpandEnv(cfg.Gist.Token), cfg.ProxyPrefix)
	if err := gc.PushResults(cfg.Gist.GistID, cfg.CF.OutputFile, results); err != nil {
		log.Fatal("gist update failed:", err)
	}

	log.Println("done, results pushed to gist. check local file at:", outputFilePath)
}