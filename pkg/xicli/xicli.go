// Package xicli provides an interface to the xicli command-line tool
package xicli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// Executor handles execution of xicli commands.
type Executor struct {
	xicliPath string
}

// NewExecutor creates a new xicli executor.
func NewExecutor(path string) *Executor {
	return &Executor{
		xicliPath: path,
	}
}

// RaidInfo represents the output of 'xicli raid show --extended --format json'.
type RaidInfo struct {
	Name              string          `json:"name"`
	Active            bool            `json:"active"`
	Config            bool            `json:"config"`
	Level             string          `json:"level"`
	State             []string        `json:"state"`
	Devices           [][]interface{} `json:"devices"`
	DevicesHealth     []string        `json:"devices_health"`
	DevicesWear       []string        `json:"devices_wear"`
	GroupSize         int             `json:"group_size"`
	MemoryUsageMB     int             `json:"memory_usage_mb"`
	MemoryPreallocMB  int             `json:"memory_prealloc_mb,omitempty"`
	MemoryLimitMB     int             `json:"memory_limit_mb,omitempty"`
	BlockSize         int             `json:"block_size"`
	StripSize         int             `json:"strip_size"`
	Size              string          `json:"size"`
	UUID              string          `json:"uuid"`
	Serials           []string        `json:"serials"`
	Sparepool         string          `json:"sparepool"`
	CPUAllowed        []int           `json:"cpu_allowed,omitempty"`
	InitPrio          int             `json:"init_prio,omitempty"`
	ReconPrio         int             `json:"recon_prio,omitempty"`
	RestripePrio      int             `json:"restripe_prio,omitempty"`
	SDCPrio           int             `json:"sdc_prio,omitempty"`
	RequestLimit      int             `json:"request_limit,omitempty"`
	SchedEnabled      int             `json:"sched_enabled,omitempty"`
	AdaptiveMerge     int             `json:"adaptive_merge,omitempty"`
	MergeReadEnabled  int             `json:"merge_read_enabled,omitempty"`
	MergeWriteEnabled int             `json:"merge_write_enabled,omitempty"`
}

// LicenseInfo represents the output of 'xicli license show --format json'.
type LicenseInfo struct {
	Status        string `json:"status"`
	Type          string `json:"type"`
	Disks         string `json:"disks"`
	DisksInUse    string `json:"disks_in_use"`
	Levels        string `json:"levels"`
	Created       string `json:"created"`
	Expired       string `json:"expired"`
	KernelVersion string `json:"Kernel version"`
	HWKey         string `json:"hwkey"`
}

// MailInfo represents the output of 'xicli mail show --format json'.
type MailInfo struct {
	Recipient string `json:"recipient"`
	Level     string `json:"level"`
}

// PoolInfo represents the output of 'xicli pool show --format json'.
type PoolInfo struct {
	Name    string `json:"name"`
	Devices string `json:"devices"`
	Serials string `json:"serials"`
	Sizes   string `json:"sizes"`
	State   string `json:"state"`
}

// SettingsAuth represents the output of 'xicli settings auth show --format json'.
type SettingsAuth struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// SettingsCluster represents the output of 'xicli settings cluster show --format json'.
type SettingsCluster struct {
	PoolAutoactivate int `json:"pool_autoactivate"`
	RaidAutostart    int `json:"raid_autostart"`
}

// SettingsEula represents the output of 'xicli settings eula show --format json'.
type SettingsEula struct {
	Status string `json:"status"`
}

// SettingsFaultyCount represents the output of 'xicli settings faulty-count show --format json'.
type SettingsFaultyCount struct {
	FaultyCountThreshold int `json:"faulty_count_threshold"`
}

// SettingsLog represents the output of 'xicli settings log show --format json'.
type SettingsLog struct {
	Level        string `json:"level"`
	LogDirectory string `json:"log directory"`
}

// SettingsMail represents the output of 'xicli settings mail show --format json'.
type SettingsMail struct {
	PollingInterval         int `json:"polling_interval"`
	ProgressPollingInterval int `json:"progress_polling_interval"`
}

// SettingsPool represents the output of 'xicli settings pool show --format json'.
type SettingsPool struct {
	ReplaceDelay int `json:"replace_delay"`
}

// SettingsScanner represents the output of 'xicli settings scanner show --format json'.
type SettingsScanner struct {
	LoedEnabled            int `json:"loed_enabled"`
	ScannerPollingInterval int `json:"scanner_polling_interval"`
	SmartPollingInterval   int `json:"smart_polling_interval"`
}

// GetRaidInfo executes 'xicli raid show --format json' (without --extended).
func (x *Executor) GetRaidInfo() (map[string]RaidInfo, error) {
	output, err := x.executeCommand("raid", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli raid show: %w", err)
	}

	var raids map[string]RaidInfo

	err = json.Unmarshal(output, &raids)
	if err != nil {
		return nil, fmt.Errorf("failed to parse raid info: %w", err)
	}

	return raids, nil
}

// GetRaidInfoExtended executes 'xicli raid show --extended --format json'.
func (x *Executor) GetRaidInfoExtended() (map[string]RaidInfo, error) {
	output, err := x.executeCommand("raid", "show", "--extended", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli raid show --extended: %w", err)
	}

	var raids map[string]RaidInfo

	err = json.Unmarshal(output, &raids)
	if err != nil {
		return nil, fmt.Errorf("failed to parse raid info: %w", err)
	}

	return raids, nil
}

// GetFaultyCount executes 'xicli drive faulty-count show --format json'.
func (x *Executor) GetFaultyCount() (map[string]int, error) {
	output, err := x.executeCommand("drive", "faulty-count", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli drive faulty-count show: %w", err)
	}

	var faultyCounts map[string]int

	err = json.Unmarshal(output, &faultyCounts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse faulty count: %w", err)
	}

	return faultyCounts, nil
}

// GetLicenseInfo executes 'xicli license show --format json'.
func (x *Executor) GetLicenseInfo() (*LicenseInfo, error) {
	output, err := x.executeCommand("license", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli license show: %w", err)
	}

	var license LicenseInfo

	err = json.Unmarshal(output, &license)
	if err != nil {
		return nil, fmt.Errorf("failed to parse license info: %w", err)
	}

	return &license, nil
}

// GetMailInfo executes 'xicli mail show --format json'.
func (x *Executor) GetMailInfo() (*MailInfo, error) {
	output, err := x.executeCommand("mail", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli mail show: %w", err)
	}

	var mail MailInfo

	err = json.Unmarshal(output, &mail)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mail info: %w", err)
	}

	return &mail, nil
}

// GetPoolInfo executes 'xicli pool show --format json'.
func (x *Executor) GetPoolInfo() (*PoolInfo, error) {
	output, err := x.executeCommand("pool", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli pool show: %w", err)
	}

	var pool PoolInfo

	err = json.Unmarshal(output, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool info: %w", err)
	}

	return &pool, nil
}

// GetSettingsAuth executes 'xicli settings auth show --format json'.
func (x *Executor) GetSettingsAuth() (*SettingsAuth, error) {
	output, err := x.executeCommand("settings", "auth", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli settings auth show: %w", err)
	}

	var settings SettingsAuth

	err = json.Unmarshal(output, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings auth: %w", err)
	}

	return &settings, nil
}

// GetSettingsCluster executes 'xicli settings cluster show --format json'.
func (x *Executor) GetSettingsCluster() (*SettingsCluster, error) {
	output, err := x.executeCommand("settings", "cluster", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli settings cluster show: %w", err)
	}

	var settings SettingsCluster

	err = json.Unmarshal(output, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings cluster: %w", err)
	}

	return &settings, nil
}

// GetSettingsEula executes 'xicli settings eula show --format json'.
func (x *Executor) GetSettingsEula() (*SettingsEula, error) {
	output, err := x.executeCommand("settings", "eula", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli settings eula show: %w", err)
	}

	var settings SettingsEula

	err = json.Unmarshal(output, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings eula: %w", err)
	}

	return &settings, nil
}

// GetSettingsFaultyCount executes 'xicli settings faulty-count show --format json'.
func (x *Executor) GetSettingsFaultyCount() (*SettingsFaultyCount, error) {
	output, err := x.executeCommand("settings", "faulty-count", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli settings faulty-count show: %w", err)
	}

	var settings SettingsFaultyCount

	err = json.Unmarshal(output, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings faulty-count: %w", err)
	}

	return &settings, nil
}

// GetSettingsLog executes 'xicli settings log show --format json'.
func (x *Executor) GetSettingsLog() (*SettingsLog, error) {
	output, err := x.executeCommand("settings", "log", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli settings log show: %w", err)
	}

	var settings SettingsLog

	err = json.Unmarshal(output, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings log: %w", err)
	}

	return &settings, nil
}

// GetSettingsMail executes 'xicli settings mail show --format json'.
func (x *Executor) GetSettingsMail() (*SettingsMail, error) {
	output, err := x.executeCommand("settings", "mail", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli settings mail show: %w", err)
	}

	var settings SettingsMail

	err = json.Unmarshal(output, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings mail: %w", err)
	}

	return &settings, nil
}

// GetSettingsPool executes 'xicli settings pool show --format json'.
func (x *Executor) GetSettingsPool() (*SettingsPool, error) {
	output, err := x.executeCommand("settings", "pool", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli settings pool show: %w", err)
	}

	var settings SettingsPool

	err = json.Unmarshal(output, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings pool: %w", err)
	}

	return &settings, nil
}

// GetSettingsScanner executes 'xicli settings scanner show --format json'.
func (x *Executor) GetSettingsScanner() (*SettingsScanner, error) {
	output, err := x.executeCommand("settings", "scanner", "show", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli settings scanner show: %w", err)
	}

	var settings SettingsScanner

	err = json.Unmarshal(output, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings scanner: %w", err)
	}

	return &settings, nil
}

// executeCommand executes an xicli command with context and returns the output.
func (x *Executor) executeCommand(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// #nosec G204 - xicliPath is configured via flag, not user input
	cmd := exec.CommandContext(ctx, x.xicliPath, args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute xicli command: %w", err)
	}

	return output, nil
}
