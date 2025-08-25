package models

// GistContent 是上传到 Gist 的 JSON 文件的完整结构体
type GistContent struct {
	Timestamp string         `json:"timestamp"`
	Results   []DeviceResult `json:"results"`
}

// DeviceResult 代表单条测速结果
// [修改] 调整 json 标签以从最终 json 中排除某些字段
type DeviceResult struct {
	Device    string  `json:"-"` // 在 JSON 序列化时忽略此字段
	Operator  string  `json:"-"` // 在 JSON 序列化时忽略此字段
	IP        string  `json:"ip"`
	LatencyMs int     `json:"latency_ms"`
	LossPct   float64 `json:"loss_pct"`
	DLMBps    float64 `json:"dl_mbps"`
	Region    string  `json:"region"`
}