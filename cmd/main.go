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
const configDir = "config"
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
		)
		if err := inst.InstallOrUpdate(); err != nil {
			log.Fatalf("install/update cf failed: %v", err)
		}
	}

	// [修改] 构建输出文件的完整路径
    outputFilePath := filepath.Join(configDir, cfg.CF.OutputFile)

	// [修改] 将输出文件的完整路径传递给 Tester
	cf := tester.NewCFSpeedTester(cfg.CF.Binary, outputFilePath, cfg.DeviceName, cfg.CF.Args)
	results, err := cf.Run()
	if err != nil {
		log.Fatal("speed test failed:", err)
	}

	gc := gist.NewClient(os.ExpandEnv(cfg.Gist.Token), cfg.ProxyPrefix)
	// [修改] Gist 推送的文件名保持不变，但内容来自新的路径
	if err := gc.PushResults(cfg.Gist.GistID, cfg.CF.OutputFile, results); err != nil {
		log.Fatal("gist update failed:", err)
	}

	log.Println("done, results pushed to gist. check local file at:", outputFilePath)
}