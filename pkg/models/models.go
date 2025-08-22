package models

// [最终修改] 与 CFST 工具的 CSV 输出完全对应
type DeviceResult struct {
	Device    string  `json:"device"`
	IP        string  `json:"ip"`
	LatencyMs int     `json:"latency_ms"`
	LossPct   float64 `json:"loss_pct"` // [新增] 从 CSV 获取
	DLMBps    float64 `json:"dl_mbps"`    // [FIX] Renamed field to reflect MB/s unit
}