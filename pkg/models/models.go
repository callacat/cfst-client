package models

// GistContent 是上传到 Gist 的 JSON 文件的完整结构体
type GistContent struct {
	Timestamp string         `json:"timestamp"`
	Results   []DeviceResult `json:"results"`
}

// DeviceResult 代表单条测速结果
// [修改] 新增了 Region 字段
type DeviceResult struct {
	Device    string  `json:"device"`
	Operator  string  `json:"operator"`
	IP        string  `json:"ip"`
	LatencyMs int     `json:"latency_ms"`
	LossPct   float64 `json:"loss_pct"`
	DLMBps    float64 `json:"dl_mbps"`
	Region    string  `json:"region"` // 新增地区码字段
}