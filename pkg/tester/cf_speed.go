package tester

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"cfst-client/pkg/models"
)

type CFSpeedTester struct {
	bin        string
	args       []string
	outputFile string // [新增]
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

// [最终重构] 执行命令并解析 CSV 结果文件
func (c *CFSpeedTester) Run() ([]models.DeviceResult, error) {
	// 将 -o 参数和文件名附加到参数列表
	cmdArgs := append(c.args, "-o", c.outputFile)

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

	// 创建 CSV 读取器
	reader := csv.NewReader(file)
	// 读取 CSV 头部，并忽略
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv header: %w", err)
	}

	var results []models.DeviceResult

	// 逐行读取 CSV 内容
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading csv record: %w", err)
		}
		if len(record) < 6 {
			continue // 跳过格式不正确的行
		}

		// 解析 CSV 字段
		// 格式: IP, 已发送, 已接收, 丢包率, 平均延迟, 下载速度(MB/s)
		ip := record[0]
		loss, _ := strconv.ParseFloat(record[3], 64)
		latency, _ := strconv.ParseFloat(record[4], 64)
		speed, _ := strconv.ParseFloat(record[5], 64)

		// 填充我们的标准模型
		results = append(results, models.DeviceResult{
			Device:    c.deviceName,
			Operator:  "", // CSV 中无此信息，留空或从配置中获取
			IP:        ip,
			LatencyMs: int(latency),
			LossPct:   loss,
			DLMbps:    speed * 8, // MB/s 转换为 Mbps
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid results parsed from csv file")
	}

	return results, nil
}