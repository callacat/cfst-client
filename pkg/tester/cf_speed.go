package tester

import (
	"encoding/csv"
	"fmt"
	"io"
	"log" // [新增] 导入 log 包
	"os"
	"os/exec"
	"strconv"
	"strings" // [新增] 导入 strings 包

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
	// 将 -o 参数和文件名附加到参数列表
	cmdArgs := append(c.args, "-o", c.outputFile)

	// [核心改进] 打印将要执行的完整命令，便于调试
	fullCommand := fmt.Sprintf("%s %s", c.bin, strings.Join(cmdArgs, " "))
	log.Printf("Executing command: %s", fullCommand)

	// 执行 CloudflareSpeedTest 命令
	cmd := exec.Command(c.bin, cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec error: %v, output: %s", err, string(out))
	}

	// 打开生成的 CSV 文件
	file, err := os.Open(c.outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open result file '%s': %w", c.outputFile, err)
	}
	defer file.Close()
	
    // ... (后续的 CSV 解析逻辑保持不变) ...
	reader := csv.NewReader(file)
	_, err = reader.Read()
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
		return nil, fmt.Errorf("no valid results parsed from csv file, raw output was: %s", string(out))
	}

	return results, nil
}