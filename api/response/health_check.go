package response

// HealthCheckResponse 健康检查接口的响应结构体
type HealthCheckResponse struct {
	// Status 服务状态，例如 "ok"
	Status string `json:"status"`
	// ResponseMs 响应耗时，单位毫秒
	ResponseMs string `json:"response_ms"`
	// System 系统资源使用情况
	System SystemStats `json:"system"`
	// Process 当前进程的运行状态
	Process ProcessStats `json:"process"`
}

// SystemStats 系统级别的资源使用情况
type SystemStats struct {
	// CPUPercent CPU 使用率（%）
	CPUPercent string `json:"cpu_percent"`
	// MemoryTotalMB 内存总量（MB）
	MemoryTotalMB string `json:"memory_total"`
	// MemoryUsedMB 已使用内存（MB）
	MemoryUsedMB string `json:"memory_used"`
	// MemoryPercent 内存使用率（%）
	MemoryPercent string `json:"memory_percent"`
	// DiskTotalGB 磁盘总量（GB）
	DiskTotalGB string `json:"disk_total"`
	// DiskUsedGB 已使用磁盘（GB）
	DiskUsedGB string `json:"disk_used"`
	// DiskPercent 磁盘使用率（%）
	DiskPercent string `json:"disk_percent"`
}

// ProcessStats 当前 Go 进程的资源使用情况
type ProcessStats struct {
	// CPUPercent 进程 CPU 使用率（%）
	CPUPercent string `json:"cpu_percent"`
	// MemoryRSSMB 进程常驻内存（MB）
	MemoryRSSMB string `json:"memory_rss_mb"`
	// Goroutines 当前运行的 goroutine 数
	Goroutines string `json:"goroutines"`
	// GoHeapAllocMB Go 堆分配的内存（MB）
	GoHeapAllocMB string `json:"go_heap_alloc_mb"`
}
