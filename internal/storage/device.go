package storage

import (
	"encoding/json"
	"encrp/internal/config"
	"encrp/internal/errors"
)

type Device struct {
	hostID          string
	os              string
	platform        string
	platformVersion string
	platformFamily  string
	architecture    string
	cpuCores        int
	hostname        string
	totalRAM        string
	totalDisk       string
}

type device struct {
	HostID          string `json:"host_id"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	PlatformFamily  string `json:"platform_family"`
	Architecture    string `json:"architecture"`
	CPUCores        int    `json:"cpu_cores"`
	Hostname        string `json:"hostname"`
	TotalRAM        string `json:"total_ram"`
	TotalDisk       string `json:"total_disk"`
}

func NewDeviceFromConfig(c *config.Device) *Device {
	d := &Device{}
	d.SetFromConfig(c)
	return d
}

func (d *Device) HostID() string          { return d.hostID }
func (d *Device) OS() string              { return d.os }
func (d *Device) Platform() string        { return d.platform }
func (d *Device) PlatformVersion() string { return d.platformVersion }
func (d *Device) PlatformFamily() string  { return d.platformFamily }
func (d *Device) Architecture() string    { return d.architecture }
func (d *Device) CPUCores() int           { return d.cpuCores }
func (d *Device) Hostname() string        { return d.hostname }
func (d *Device) TotalRAM() string        { return d.totalRAM }
func (d *Device) TotalDisk() string       { return d.totalDisk }

func (d *Device) SetFromConfig(c *config.Device) {
	if c == nil {
		return
	}
	d.hostID = c.HostID
	d.os = c.OS
	d.platform = c.Platform
	d.platformVersion = c.PlatformVersion
	d.platformFamily = c.PlatformFamily
	d.architecture = c.Architecture
	d.cpuCores = c.CPUCores
	d.hostname = c.Hostname
	d.totalRAM = c.TotalRAM
	d.totalDisk = c.TotalDisk
}

func (d *Device) UnmarshalJSON(data []byte) error {
	temp := &device{}
	if err := json.Unmarshal(data, temp); err != nil {
		return errors.Wrap(err, "Device.UnmarshalJSON()", "Failed to unmarshal Device")
	}
	d.hostID = temp.HostID
	d.os = temp.OS
	d.platform = temp.Platform
	d.platformVersion = temp.PlatformVersion
	d.platformFamily = temp.PlatformFamily
	d.architecture = temp.Architecture
	d.cpuCores = temp.CPUCores
	d.hostname = temp.Hostname
	d.totalRAM = temp.TotalRAM
	d.totalDisk = temp.TotalDisk
	return nil
}

func (d *Device) MarshalJSON() ([]byte, error) {
	temp := &device{
		HostID:          d.HostID(),
		OS:              d.OS(),
		Platform:        d.Platform(),
		PlatformVersion: d.PlatformVersion(),
		PlatformFamily:  d.PlatformFamily(),
		Architecture:    d.Architecture(),
		CPUCores:        d.CPUCores(),
		Hostname:        d.Hostname(),
		TotalRAM:        d.TotalRAM(),
		TotalDisk:       d.TotalDisk(),
	}
	data, err := json.Marshal(temp)
	if err != nil {
		return nil, errors.Wrap(err, "Device.MarshalJSON()", "Failed to marshal Device")
	}
	return data, nil
}
