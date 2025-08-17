package response

type HealthCheckResponse struct {
	Status     string       `json:"status"`
	ResponseMs int64        `json:"response_ms"`
	System     SystemStats  `json:"system"`
	Process    ProcessStats `json:"process"`
}

type SystemStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryTotalMB uint64  `json:"memory_total"`
	MemoryUsedMB  uint64  `json:"memory_used"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskTotalGB   uint64  `json:"disk_total"`
	DiskUsedGB    uint64  `json:"disk_used"`
	DiskPercent   float64 `json:"disk_percent"`
}

type ProcessStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryRSSMB   uint64  `json:"memory_rss_mb"`
	Goroutines    int     `json:"goroutines"`
	GoHeapAllocMB uint64  `json:"go_heap_alloc_mb"`
}
