package service

import (
	"feedback/api/response"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

func HealthCheck() response.HealthCheckResponse {
	start := time.Now() // 记录开始时间

	// 整机资源
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil || len(cpuPercent) == 0 {
		cpuPercent = []float64{0}
	}
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		vmStat = &mem.VirtualMemoryStat{}
	}
	diskStat, err := disk.Usage("/")
	if err != nil {
		diskStat = &disk.UsageStat{}
	}

	// 当前进程资源
	proc, err := process.NewProcess(int32(os.Getpid()))
	var procCPU float64
	var procMem *process.MemoryInfoStat
	if err == nil {
		procCPU, _ = proc.CPUPercent()
		procMem, _ = proc.MemoryInfo()
	} else {
		procCPU = 0
		procMem = &process.MemoryInfoStat{}
	}

	// Go runtime 信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 响应耗时
	duration := time.Since(start).Milliseconds()

	return response.HealthCheckResponse{
		Status:     "ok",
		ResponseMs: fmt.Sprintf("%d ms", duration),
		System: response.SystemStats{
			CPUPercent:    fmt.Sprintf("%.5f%%", cpuPercent[0]),
			MemoryTotalMB: fmt.Sprintf("%d MB", vmStat.Total/1024/1024),
			MemoryUsedMB:  fmt.Sprintf("%d MB", vmStat.Used/1024/1024),
			MemoryPercent: fmt.Sprintf("%.5f%%", vmStat.UsedPercent),
			DiskTotalGB:   fmt.Sprintf("%d GB", diskStat.Total/1024/1024/1024),
			DiskUsedGB:    fmt.Sprintf("%d GB", diskStat.Used/1024/1024/1024),
			DiskPercent:   fmt.Sprintf("%.5f%%", diskStat.UsedPercent),
		},
		Process: response.ProcessStats{
			CPUPercent:    fmt.Sprintf("%.5f%%", procCPU),
			MemoryRSSMB:   fmt.Sprintf("%d MB", procMem.RSS/1024/1024),
			Goroutines:    fmt.Sprintf("%d", runtime.NumGoroutine()),
			GoHeapAllocMB: fmt.Sprintf("%d MB", m.HeapAlloc/1024/1024),
		},
	}
}
