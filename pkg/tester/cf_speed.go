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

// CFSpeedTester holds the configuration for a speed test run.
type CFSpeedTester struct {
	bin          string
	args         []string
	outputFile   string
	deviceName   string
	lineOperator string
}

// NewCFSpeedTester creates a new instance of CFSpeedTester.
func NewCFSpeedTester(bin, outputFile, deviceName, lineOperator string, args []string) *CFSpeedTester {
	return &CFSpeedTester{
		bin:          bin,
		args:         args,
		outputFile:   outputFile,
		deviceName:   deviceName,
		lineOperator: lineOperator,
	}
}

// Run executes the CloudflareSpeedTest command and parses the results.
func (c *CFSpeedTester) Run() ([]models.DeviceResult, error) {
	_ = os.Remove(c.outputFile)

	cmdArgs := append(c.args, "-o", c.outputFile)
	fullCommand := fmt.Sprintf("%s %s", c.bin, strings.Join(cmdArgs, " "))
	log.Printf("Executing command: %s", fullCommand)

	cmd := exec.Command(c.bin, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}
	log.Println("CloudflareSpeedTest finished successfully.")

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
		// [修改] 现在要求至少有 7 列数据
		if len(record) < 7 {
			continue
		}

		ip := record[0]
		loss, _ := strconv.ParseFloat(record[3], 64)
		latency, _ := strconv.ParseFloat(record[4], 64)
		speed, _ := strconv.ParseFloat(record[5], 64)
		region := record[6] // [新增] 获取地区码

		results = append(results, models.DeviceResult{
			Device:    c.deviceName,
			Operator:  c.lineOperator,
			IP:        ip,
			LatencyMs: int(latency),
			LossPct:   loss,
			DLMBps:    speed,
			Region:    region, // [新增] 填充地区码字段
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid results parsed from csv file")
	}

	return results, nil
}