package models

// [最终修改] 与 CFST 工具的 CSV 输出完全对应
type DeviceResult struct {
	Device    string  `json:"device"`
	Operator  string  `json:"operator"` // 注意：CSV 中无此字段，将由其他方式填充
	IP        string  `json:"ip"`
	LatencyMs int     `json:"latency_ms"`
	LossPct   float64 `json:"loss_pct"` // [新增] 从 CSV 获取
	DLMbps    float64 `json:"dl_mbps"`
}