package config

import (
	"encrp/internal/logger"
	"fmt"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"runtime"
)

type Device struct {
	HostID          string
	OS              string
	Platform        string
	PlatformVersion string
	PlatformFamily  string
	Architecture    string
	CPUCores        int
	Hostname        string
	TotalRAM        string
	TotalDisk       string
}

func newDevice() *Device {
	device := &Device{}
	device.CPUCores = runtime.NumCPU()
	device.OS = runtime.GOOS
	device.Architecture = runtime.GOARCH

	hostInfo, err := host.Info()
	if err == nil {
		device.HostID = hostInfo.HostID
		device.Platform = hostInfo.Platform
		device.PlatformFamily = hostInfo.PlatformFamily
		device.PlatformVersion = hostInfo.PlatformVersion
		device.Hostname = hostInfo.Hostname
	} else {
		logger.Errf("Config.LastDevice()", "failed to get host info: %v", err)
	}

	vmem, err := mem.VirtualMemory()
	if err == nil {
		device.TotalRAM = fmt.Sprintf("%.2f GB", float64(vmem.Total)/1024/1024/1024)
	} else {
		logger.Errf("Config.LastDevice()", "failed to get virtual memory: %v", err)
	}

	diskPath := "/"
	if device.OS == "windows" {
		diskPath = "C:\\"
	}
	diskStat, err := disk.Usage(diskPath)
	if err == nil {
		device.TotalDisk = fmt.Sprintf("%.2f GB", float64(diskStat.Total)/1024/1024/1024)
	} else {
		logger.Errf("Config.LastDevice()", "failed to get disk info: %v", err)
	}
	return device
}
