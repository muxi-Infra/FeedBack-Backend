package service

import (
	"feedback/api/response"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"os"
	"runtime"
	"time"
)

func HealthCheck() response.HealthCheckResponse {
	start := time.Now() // 记录开始时间

	// ==== 整机资源 ====
	cpuPercent, _ := cpu.Percent(0, false)
	vmStat, _ := mem.VirtualMemory()
	diskStat, _ := disk.Usage("/")

	// ==== 当前进程资源 ====
	proc, _ := process.NewProcess(int32(os.Getpid()))
	procCPU, _ := proc.CPUPercent()
	procMem, _ := proc.MemoryInfo()

	// Go runtime 信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 响应耗时
	duration := time.Since(start).Milliseconds()

	return response.HealthCheckResponse{
		Status:     "ok",
		ResponseMs: duration,
		System: response.SystemStats{
			CPUPercent:    cpuPercent[0],
			MemoryTotalMB: vmStat.Total / 1024 / 1024,
			MemoryUsedMB:  vmStat.Used / 1024 / 1024,
			MemoryPercent: vmStat.UsedPercent,
			DiskTotalGB:   diskStat.Total / 1024 / 1024 / 1024,
			DiskUsedGB:    diskStat.Used / 1024 / 1024 / 1024,
			DiskPercent:   diskStat.UsedPercent,
		},
		Process: response.ProcessStats{
			CPUPercent:    procCPU,
			MemoryRSSMB:   procMem.RSS / 1024 / 1024,
			Goroutines:    runtime.NumGoroutine(),
			GoHeapAllocMB: m.HeapAlloc / 1024 / 1024,
		},
	}

}
