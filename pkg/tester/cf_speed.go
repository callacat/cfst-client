package tester

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"cfst-client/pkg/models"
)

// ... (CFSpeedTester 结构体和 NewCFSpeedTester 函数保持不变) ...
type CFSpeedTester struct {
	bin        string
	args       []string
	outputFile string
	deviceName string
}

func NewCFSpeedTester(bin, outputFile, deviceName string, args []string) *CFSpeedTester {
	return &CFSpeedTester{
		bin:        bin,
		args:       args,
		outputFile: outputFile,
		deviceName: deviceName,
	}
}


func (c *CFSpeedTester) Run() ([]models.DeviceResult, error) {
	cmdArgs := append(c.args, "-o", c.outputFile)
	fullCommand := fmt.Sprintf("%s %s", c.bin, strings.Join(cmdArgs, " "))
	log.Printf("Executing command: %s", fullCommand)

	// [修正] 恢复为简单的命令执行，移除模拟输入的逻辑
	cmd := exec.Command(c.bin, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}
	log.Println("CloudflareSpeedTest finished successfully.")


	// --- 后续的 CSV 解析逻辑保持不变 ---
	file, err := os.Open(c.outputFile)
    // ... (后续代码完全不变) ...
	if err != nil {
		return nil, fmt.Errorf("failed to open result file '%s': %w", c.outputFile, err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	_, err = reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("result file is empty or missing header: %s", c.outputFile)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read csv header: %w", err)
	}
	var results []models.DeviceResult
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading csv record: %w", err)
		}
		if len(record) < 6 {
			continue
		}
		ip := record[0]
		loss, _ := strconv.ParseFloat(record[3], 64)
		latency, _ := strconv.ParseFloat(record[4], 64)
		speed, _ := strconv.ParseFloat(record[5], 64)
		results = append(results, models.DeviceResult{
			Device:    c.deviceName,
			Operator:  "",
			IP:        ip,
			LatencyMs: int(latency),
			LossPct:   loss,
			DLMbps:    speed * 8,
		})
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no valid results parsed from csv file")
	}
	return results, nil
}