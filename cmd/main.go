package main

import (
	"log"
	"os"

	"cfst-client/pkg/config"
	"cfst-client/pkg/gist"
	"cfst-client/pkg/installer"
	"cfst-client/pkg/tester"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatal("load config:", err)
	}

	// 自动更新 cf binary
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

	// 测速 (传入 device_name)
	cf := tester.NewCFSpeedTester(cfg.CF.Binary, cfg.CF.Args, cfg.DeviceName) // [修改]
	results, err := cf.Run()
	if err != nil {
		log.Fatal("speed test failed:", err)
	}

	// 上传 Gist
	gc := gist.NewClient(os.ExpandEnv(cfg.Gist.Token), cfg.ProxyPrefix)
	if err := gc.PushResults(cfg.Gist.GistID, cfg.CF.OutputFile, results); err != nil {
		log.Fatal("gist update failed:", err)
	}

	log.Println("done, pushed file:", cfg.CF.OutputFile)
}