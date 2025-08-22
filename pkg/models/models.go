package models

// [最终修改] 与 CFST 工具的 CSV 输出完全对应
type DeviceResult struct {
	Device    string  `json:"device"`
	Operator  string  `json:"operator"` // [FIX] Added the missing Operator field
	IP        string  `json:"ip"`
	LatencyMs int     `json:"latency_ms"`
	LossPct   float64 `json:"loss_pct"` 
	DLMBps    float64 `json:"dl_mbps"`
}