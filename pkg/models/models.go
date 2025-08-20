package models

// [修改] DeviceResult 结构与主控兼容，补全字段
type DeviceResult struct {
	Device    string  `json:"device"`
	Operator  string  `json:"operator"`
	IP        string  `json:"ip"`
	LatencyMs int     `json:"latency_ms"`
	JitterMs  int     `json:"jitter_ms"`   // [新增]
	LossPct   float64 `json:"loss_pct"`    // [新增]
	DLMbps    float64 `json:"dl_mbps"`
}