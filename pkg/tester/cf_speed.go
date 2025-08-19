package tester

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "cfst-client/pkg/models"
)

type CFSpeedTester struct {
    bin    string
    args   []string
    output string
}

func NewCFSpeedTester(bin string, args []string, outputFile string) *CFSpeedTester {
    return &CFSpeedTester{bin: bin, args: args, output: outputFile}
}

func (c *CFSpeedTester) Run() ([]models.DeviceResult, error) {
    out, err := exec.Command(c.bin, c.args...).CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("exec error: %v, output: %s", err, out)
    }

    var raw []struct {
        ISP       string  `json:"isp"`
        Region    string  `json:"region"`
        RTT       float64 `json:"rtt"`
        Bandwidth float64 `json:"bandwidth"`
    }
    if err := json.Unmarshal(out, &raw); err != nil {
        return nil, fmt.Errorf("json unmarshal: %w", err)
    }

    var res []models.DeviceResult
    for _, r := range raw {
        res = append(res, models.DeviceResult{
            Device:    "cf-speed",
            Operator:  r.ISP,
            IP:        r.Region,
            LatencyMs: int(r.RTT),
            DLMbps:    r.Bandwidth * 8 / 1e6,
        })
    }
    return res, nil
}
