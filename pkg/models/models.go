package models

// DeviceResult 结构与主控兼容
type DeviceResult struct {
    Device    string  `json:"device"`
    Operator  string  `json:"operator"`
    IP        string  `json:"ip"`
    LatencyMs int     `json:"latency_ms"`
    DLMbps    float64 `json:"dl_mbps"`
}
