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

	// 打印将要执行的完整命令
	fullCommand := fmt.Sprintf("%s %s", c.bin, strings.Join(cmdArgs, " "))
	log.Printf("Executing command: %s", fullCommand)

	// [核心修改]
	// 1. 创建命令对象
	cmd := exec.Command(c.bin, cmdArgs...)
	// 2. 将子程序的标准输出和标准错误直接连接到当前程序
	//    这样，子程序的所有输出都会实时打印到 Docker 日志中
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 3. 使用 Run() 执行命令。它会等待命令完成，但不会捕获输出。
	//    如果命令执行出错（例如，返回非零退出码），它会返回一个 error。
	err := cmd.Run()
	if err != nil {
		// 由于输出已经实时打印，我们这里只返回一个更简洁的错误
		return nil, fmt.Errorf("command execution failed: %w", err)
	}
	log.Println("CloudflareSpeedTest finished successfully.")


	// --- 后续的 CSV 解析逻辑保持不变 ---

	file, err := os.Open(c.outputFile)
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