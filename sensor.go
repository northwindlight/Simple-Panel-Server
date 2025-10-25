// Updated sensor.go (only GetSystemInfo function modified)
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/northwindlight/cputemp"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/sirupsen/logrus"
)

type TotalDiskInfo struct {
	MountPoint   string
	TotalGB      float64
	UsedGB       float64
	UsagePercent int
}

type TotalMem struct {
	TotalMB      int
	UsedMB       int
	UsagePercent int
}

// SystemInfo holds system information for /info endpoint
type SystemInfo struct {
	OS            string  `json:"os"`
	Platform      string  `json:"platform"`
	Kernel        string  `json:"kernel"`
	UptimeSeconds uint64  `json:"uptime_seconds"`
	CPUModel      string  `json:"cpu_model"`
	CPUSpecs      string  `json:"cpu_specs"` // e.g., "4 Cores / 8 Threads"
	MemTotalGB    float64 `json:"mem_total_gb"`
	DiskTotalGB   float64 `json:"disk_total_gb"`
}

func getCPUUsage() int {
	usage, _ := cpu.Percent(2*time.Second, false)
	if len(usage) > 0 {
		return int(usage[0])
	}
	return 0
}

func getCPUTemperature() int {
	temp, err := cputemp.GetCPUTemperature()
	if err != nil {
		logrus.Error(err)
	}
	return int(temp)
}

func getMemoryUsage() (TotalMem, error) {
	mem, err := mem.VirtualMemory()
	if err != nil {
		return TotalMem{}, err
	}
	return TotalMem{
		TotalMB:      int(mem.Total / 1024 / 1024),
		UsedMB:       int(mem.Used / 1024 / 1024),
		UsagePercent: int(mem.Used * 100 / mem.Total),
	}, nil
}

func GetTotalDiskUsage() (TotalDiskInfo, error) {
	partitions, err := disk.Partitions(false) // false: 排除虚拟分区
	if err != nil {
		return TotalDiskInfo{}, fmt.Errorf("failed to get partitions: %v", err)
	}

	var totalTotal, totalUsed uint64
	const gb = 1024 * 1024 * 1024 // 1 GB = 1024^3 bytes

	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			logrus.Error("failed to get usage for %s: %v", p.Mountpoint, err)
			continue // 跳过错误的分区
		}

		totalTotal += usage.Total
		totalUsed += usage.Used
	}

	if totalTotal == 0 {
		return TotalDiskInfo{}, fmt.Errorf("no valid disk data found")
	}

	totalGB := float64(totalTotal) / float64(gb)
	usedGB := float64(totalUsed) / float64(gb)
	usagePercent := int((float64(totalUsed) / float64(totalTotal)) * 100)

	return TotalDiskInfo{
		TotalGB:      totalGB,
		UsedGB:       usedGB,
		UsagePercent: usagePercent,
	}, nil
}

// GetSystemInfo retrieves comprehensive system information
func GetSystemInfo() (SystemInfo, error) {
	// OS Info
	osInfo, err := host.Info()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("failed to get OS info: %v", err)
	}

	// Kernel
	kernel, err := host.KernelVersion()
	if err != nil {
		kernel = "unknown"
	}

	// Uptime
	uptime, err := host.Uptime()
	if err != nil {
		uptime = 0
	}

	// CPU Info
	cpuInfos, err := cpu.Info()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("failed to get CPU info: %v", err)
	}
	var cpuModel string
	var physicalCores int
	var threads int
	if len(cpuInfos) > 0 {
		cpuModel = cpuInfos[0].ModelName
		physicalCores, _ = cpu.Counts(false)
		threads, _ = cpu.Counts(true)
	} else {
		cpuModel = "unknown"
		physicalCores = 0
		threads = 0
	}
	cpuSpecs := fmt.Sprintf("%d Cores / %d Threads", physicalCores, threads)

	// Memory
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("failed to get memory info: %v", err)
	}
	memTotalGB := float64(memInfo.Total) / 1024 / 1024 / 1024

	// Disk
	diskInfo, err := GetTotalDiskUsage()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("failed to get disk info: %v", err)
	}
	diskTotalGB := diskInfo.TotalGB

	return SystemInfo{
		OS:            osInfo.Platform,
		Platform:      osInfo.PlatformVersion,
		Kernel:        kernel,
		UptimeSeconds: uptime,
		CPUModel:      cpuModel,
		CPUSpecs:      cpuSpecs,
		MemTotalGB:    memTotalGB,
		DiskTotalGB:   diskTotalGB,
	}, nil
}

func generateSystemStatus() SystemStatus {
	TotalDiskInfo, err := GetTotalDiskUsage()
	if err != nil {
		logrus.Error(err)
	}
	MemoryInfo, err := getMemoryUsage()
	if err != nil {
		logrus.Error(err)
	}
	CPUUsage := getCPUUsage()
	Temperature := getCPUTemperature()
	MemoryUsage := MemoryInfo.UsagePercent
	MemoryTotal := MemoryInfo.TotalMB
	MemoryUsed := MemoryInfo.UsedMB
	StorageUsage := TotalDiskInfo.UsagePercent
	StorageTotal := TotalDiskInfo.TotalGB
	StorageUsed := TotalDiskInfo.UsedGB
	CPUFrequency, err := GetCPUFreq()
	if err != nil {
		logrus.Error(err)
	}
	return SystemStatus{
		CPUUsage,     // 0-100%
		Temperature,  // 30-80°C
		MemoryUsage,  // 0-100%
		StorageUsage, // 0-100%
		CPUFrequency, // 2000-4000 MHz
		MemoryTotal,  // 8GB
		MemoryUsed,   // 0-8192 MB
		StorageTotal, // 500GB
		StorageUsed,  // 0-500 GB
	}
}

// formatStatusData 将系统状态格式化为前端期望的字符串格式
func formatStatusData(status SystemStatus) string {
	jsonData, err := json.Marshal(status)
	if err != nil {
		logrus.Errorf("Failed to marshal system status: %v", err)
		return ""
	}
	return string(jsonData)
}
