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
	bin        string
	args       []string
	outputFile string
	deviceName string
}

// NewCFSpeedTester creates a new instance of CFSpeedTester.
// [CORRECTED] This function signature now matches the call in main.go.
func NewCFSpeedTester(bin, outputFile, deviceName string, args []string) *CFSpeedTester {
	return &CFSpeedTester{
		bin:        bin,
		args:       args,
		outputFile: outputFile,
		deviceName: deviceName,
	}
}

// Run executes the CloudflareSpeedTest command and parses the results.
func (c *CFSpeedTester) Run() ([]models.DeviceResult, error) {
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

	// --- CSV Parsing Logic (remains the same) ---
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