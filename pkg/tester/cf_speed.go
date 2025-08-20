package tester

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"cfst-client/pkg/models"
)

type CFSpeedTester struct {
	bin        string
	args       []string
	deviceName string // [新增]
}

// [修改] 构造函数增加 deviceName
func NewCFSpeedTester(bin string, args []string, deviceName string) *CFSpeedTester {
	return &CFSpeedTester{bin: bin, args: args, deviceName: deviceName}
}

// [新增] 定义一个更完整的结构来匹配 CFST 的 -json 输出
type cfstResultItem struct {
	IP            string  `json:"ip"`
	Location      string  `json:"location"`
	ISP           string  `json:"isp"`
	AvgLatency    float64 `json:"avgLatency"`
	Jitter        float64 `json:"jitter"`
	PacketLoss    float64 `json:"packetLoss"`
	DownloadSpeed float64 `json:"downloadSpeed"` // 注意：单位是 B/s
}

// [修改] Run 方法重构以解析新格式
func (c *CFSpeedTester) Run() ([]models.DeviceResult, error) {
	out, err := exec.Command(c.bin, c.args...).CombinedOutput()
	if err != nil {
		// 检查输出是否为空，有时 CFST 即使出错也会输出一些信息
		if len(out) == 0 {
			return nil, fmt.Errorf("exec error: %v", err)
		}
		// 尝试解析可能的错误信息
		return nil, fmt.Errorf("exec error: %v, output: %s", err, string(out))
	}

	// CFST 的 -json 输出是多行 JSON 对象，需要分割处理
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var results []models.DeviceResult

	for _, line := range lines {
		var raw cfstResultItem
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			// 忽略无法解析的行，可能是标题或空行
			continue
		}

		// 将原始数据转换为我们的标准模型
		res := models.DeviceResult{
			Device:    c.deviceName, // 使用配置的设备名
			Operator:  raw.ISP,
			IP:        raw.IP,
			LatencyMs: int(raw.AvgLatency),
			JitterMs:  int(raw.Jitter),
			LossPct:   raw.PacketLoss,
			DLMbps:    raw.DownloadSpeed * 8 / 1e6, // B/s 转换为 Mbps
		}
		results = append(results, res)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid results parsed from cfst output")
	}

	return results, nil
}