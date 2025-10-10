// Package collector implements a Prometheus collector for Xiraid RAID systems
package collector

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/E4-Computer-Engineering/xiraid-exporter/pkg/xicli"
)

// Config holds configuration for which collectors are enabled.
type Config struct {
	EnableRaid         bool
	EnableRaidExtended bool
	EnableDriveFaulty  bool
	EnableLicense      bool
	EnableMail         bool
	EnablePool         bool
	EnableSettings     bool
}

// deviceRaidInfo holds RAID mapping information for a device.
type deviceRaidInfo struct {
	raidName string
	uuid     string
}

// XiraidCollector collects Xiraid metrics.
type XiraidCollector struct {
	executor *xicli.Executor
	config   Config

	// RAID metrics (basic)
	raidInfo             *prometheus.Desc
	raidConfig           *prometheus.Desc
	raidActive           *prometheus.Desc
	raidSizeBytes        *prometheus.Desc
	raidState            *prometheus.Desc
	raidMemoryUsageBytes *prometheus.Desc
	raidDeviceState      *prometheus.Desc

	// RAID extended metrics
	raidCPUAllowed          *prometheus.Desc
	raidInitPrio            *prometheus.Desc
	raidReconPrio           *prometheus.Desc
	raidRestripePrio        *prometheus.Desc
	raidSDCPrio             *prometheus.Desc
	raidRequestLimit        *prometheus.Desc
	raidSchedEnabled        *prometheus.Desc
	raidMemoryLimitBytes    *prometheus.Desc
	raidMemoryPreallocBytes *prometheus.Desc
	raidAdaptiveMerge       *prometheus.Desc
	raidMergeReadEnabled    *prometheus.Desc
	raidMergeReadMaxUsecs   *prometheus.Desc
	raidMergeReadWaitUsecs  *prometheus.Desc
	raidMergeWriteEnabled   *prometheus.Desc
	raidMergeWriteMaxUsecs  *prometheus.Desc
	raidMergeWriteWaitUsecs *prometheus.Desc
	raidDeviceHealthPercent *prometheus.Desc
	raidDeviceWearPercent   *prometheus.Desc

	// Drive metrics
	driveFaultySectorsTotal *prometheus.Desc

	// Device to RAID mapping cache
	deviceToRaidCache map[string]deviceRaidInfo

	// License metrics
	licenseValid           *prometheus.Desc
	licenseExpiryTimestamp *prometheus.Desc
	licenseDisksTotal      *prometheus.Desc
	licenseDisksInUse      *prometheus.Desc

	// Mail metrics
	mailConfigured *prometheus.Desc

	// Pool metrics
	poolInfo *prometheus.Desc

	// Settings metrics
	settingsAuthPort                *prometheus.Desc
	settingsClusterPoolAutoactivate *prometheus.Desc
	settingsClusterRaidAutostart    *prometheus.Desc
	settingsEulaAccepted            *prometheus.Desc
	settingsFaultyCountThreshold    *prometheus.Desc
	settingsMailPollingInterval     *prometheus.Desc
	settingsMailProgressInterval    *prometheus.Desc
	settingsPoolReplaceDelay        *prometheus.Desc
	settingsScannerLoedEnabled      *prometheus.Desc
	settingsScannerPollingInterval  *prometheus.Desc
	settingsScannerSmartInterval    *prometheus.Desc

	// Scrape error metrics
	scrapeError *prometheus.Desc
}

// NewXiraidCollector creates a new Xiraid collector.
func NewXiraidCollector(xicliPath string, config Config) *XiraidCollector {
	return &XiraidCollector{
		executor: xicli.NewExecutor(xicliPath),
		config:   config,

		// RAID info metric with labels for static parameters
		raidInfo: prometheus.NewDesc(
			"xiraid_raid_info",
			"RAID array information with static configuration parameters as labels",
			[]string{"raid_name", "uuid", "level", "block_size", "strip_size", "group_size", "sparepool"},
			nil,
		),

		// RAID static metrics
		raidConfig: prometheus.NewDesc(
			"xiraid_raid_config",
			"RAID array is in configuration (1=yes, 0=no)",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidActive: prometheus.NewDesc(
			"xiraid_raid_active",
			"RAID array block device is active in system (1=active, 0=inactive)",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidSizeBytes: prometheus.NewDesc(
			"xiraid_raid_size_bytes",
			"RAID array total size in bytes",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidState: prometheus.NewDesc(
			"xiraid_raid_state",
			"RAID array state (1 if in this state, 0 otherwise). "+
				"Possible states: online, initialized, initing, inconsistent, degraded, "+
				"reconstructing, offline, need_recon, read_only, restriping",
			[]string{"raid_name", "uuid", "state"},
			nil,
		),
		raidMemoryUsageBytes: prometheus.NewDesc(
			"xiraid_raid_memory_usage_bytes",
			"RAID array memory usage in bytes",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidDeviceState: prometheus.NewDesc(
			"xiraid_raid_device_state",
			"RAID device state enumeration (1=online, 2=offline, 0=none). "+
				"The 'state' label contains the actual state string",
			[]string{"raid_name", "uuid", "device", "serial", "state"},
			nil,
		),

		// RAID extended metrics
		raidCPUAllowed: prometheus.NewDesc(
			"xiraid_raid_cpu_allowed_count",
			"Number of CPUs allowed for this RAID array",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidInitPrio: prometheus.NewDesc(
			"xiraid_raid_init_priority",
			"RAID array initialization priority",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidReconPrio: prometheus.NewDesc(
			"xiraid_raid_recon_priority",
			"RAID array reconstruction priority",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidRestripePrio: prometheus.NewDesc(
			"xiraid_raid_restripe_priority",
			"RAID array restripe priority",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidSDCPrio: prometheus.NewDesc(
			"xiraid_raid_sdc_priority",
			"RAID array Silent Data Corruption scan priority",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidRequestLimit: prometheus.NewDesc(
			"xiraid_raid_request_limit",
			"RAID array request limit",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidSchedEnabled: prometheus.NewDesc(
			"xiraid_raid_scheduler_enabled",
			"RAID array scheduler enabled (1=enabled, 0=disabled)",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidMemoryLimitBytes: prometheus.NewDesc(
			"xiraid_raid_memory_limit_bytes",
			"RAID array memory limit in bytes (0=unlimited)",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidMemoryPreallocBytes: prometheus.NewDesc(
			"xiraid_raid_memory_prealloc_bytes",
			"RAID array preallocated memory in bytes",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidAdaptiveMerge: prometheus.NewDesc(
			"xiraid_raid_adaptive_merge",
			"RAID array adaptive merge enabled (1=enabled, 0=disabled)",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidMergeReadEnabled: prometheus.NewDesc(
			"xiraid_raid_merge_read_enabled",
			"RAID array merge read enabled (1=enabled, 0=disabled)",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidMergeReadMaxUsecs: prometheus.NewDesc(
			"xiraid_raid_merge_read_max_microseconds",
			"RAID array merge read maximum microseconds",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidMergeReadWaitUsecs: prometheus.NewDesc(
			"xiraid_raid_merge_read_wait_microseconds",
			"RAID array merge read wait microseconds",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidMergeWriteEnabled: prometheus.NewDesc(
			"xiraid_raid_merge_write_enabled",
			"RAID array merge write enabled (1=enabled, 0=disabled)",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidMergeWriteMaxUsecs: prometheus.NewDesc(
			"xiraid_raid_merge_write_max_microseconds",
			"RAID array merge write maximum microseconds",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidMergeWriteWaitUsecs: prometheus.NewDesc(
			"xiraid_raid_merge_write_wait_microseconds",
			"RAID array merge write wait microseconds",
			[]string{"raid_name", "uuid"},
			nil,
		),
		raidDeviceHealthPercent: prometheus.NewDesc(
			"xiraid_device_health_percent",
			"RAID device health percentage",
			[]string{"raid_name", "uuid", "device", "serial"},
			nil,
		),
		raidDeviceWearPercent: prometheus.NewDesc(
			"xiraid_device_wear_percent",
			"RAID device wear percentage",
			[]string{"raid_name", "uuid", "device", "serial"},
			nil,
		),

		// Drive metrics
		driveFaultySectorsTotal: prometheus.NewDesc(
			"xiraid_drive_faulty_sectors_total",
			"Total number of faulty sectors on drive",
			[]string{"raid_name", "uuid", "device"},
			nil,
		),

		// Initialize device to RAID mapping cache
		deviceToRaidCache: make(map[string]deviceRaidInfo),

		// License metrics
		licenseValid: prometheus.NewDesc(
			"xiraid_license_valid",
			"License is valid (1=valid, 0=invalid)",
			[]string{"status", "type"},
			nil,
		),
		licenseExpiryTimestamp: prometheus.NewDesc(
			"xiraid_license_expiry_timestamp_seconds",
			"License expiry timestamp in seconds since epoch (0 if cannot be parsed)",
			[]string{"type"},
			nil,
		),
		licenseDisksTotal: prometheus.NewDesc(
			"xiraid_license_disks_total",
			"Total number of disks allowed by license",
			[]string{"type"},
			nil,
		),
		licenseDisksInUse: prometheus.NewDesc(
			"xiraid_license_disks_in_use",
			"Number of disks currently in use",
			[]string{"type"},
			nil,
		),

		// Mail metrics
		mailConfigured: prometheus.NewDesc(
			"xiraid_mail_configured",
			"Mail notification is configured (1=configured, 0=not configured)",
			[]string{"recipient", "level"},
			nil,
		),

		// Pool metrics
		poolInfo: prometheus.NewDesc(
			"xiraid_pool_info",
			"Pool information",
			[]string{"pool_name", "devices", "serials", "sizes", "state"},
			nil,
		),

		// Settings metrics
		settingsAuthPort: prometheus.NewDesc(
			"xiraid_settings_auth_port",
			"Authentication service port number",
			[]string{"host"},
			nil,
		),
		settingsClusterPoolAutoactivate: prometheus.NewDesc(
			"xiraid_settings_cluster_pool_autoactivate",
			"Cluster pool autoactivate setting (1=enabled, 0=disabled)",
			nil,
			nil,
		),
		settingsClusterRaidAutostart: prometheus.NewDesc(
			"xiraid_settings_cluster_raid_autostart",
			"Cluster RAID autostart setting (1=enabled, 0=disabled)",
			nil,
			nil,
		),
		settingsEulaAccepted: prometheus.NewDesc(
			"xiraid_settings_eula_accepted",
			"EULA acceptance status (1=accepted, 0=not accepted)",
			[]string{"status"},
			nil,
		),
		settingsFaultyCountThreshold: prometheus.NewDesc(
			"xiraid_settings_faulty_count_threshold",
			"Faulty sector count threshold",
			nil,
			nil,
		),
		settingsMailPollingInterval: prometheus.NewDesc(
			"xiraid_settings_mail_polling_interval_seconds",
			"Mail polling interval in seconds",
			nil,
			nil,
		),
		settingsMailProgressInterval: prometheus.NewDesc(
			"xiraid_settings_mail_progress_polling_interval_seconds",
			"Mail progress polling interval in seconds",
			nil,
			nil,
		),
		settingsPoolReplaceDelay: prometheus.NewDesc(
			"xiraid_settings_pool_replace_delay_seconds",
			"Pool drive replace delay in seconds",
			nil,
			nil,
		),
		settingsScannerLoedEnabled: prometheus.NewDesc(
			"xiraid_settings_scanner_loed_enabled",
			"Scanner LOED (Latent On-disk Error Detection) enabled (1=enabled, 0=disabled)",
			nil,
			nil,
		),
		settingsScannerPollingInterval: prometheus.NewDesc(
			"xiraid_settings_scanner_polling_interval_seconds",
			"Scanner polling interval in seconds",
			nil,
			nil,
		),
		settingsScannerSmartInterval: prometheus.NewDesc(
			"xiraid_settings_scanner_smart_interval_seconds",
			"Scanner SMART polling interval in seconds",
			nil,
			nil,
		),

		// Scrape error metrics
		scrapeError: prometheus.NewDesc(
			"xiraid_scrape_error",
			"Whether an error occurred during the last scrape (1=error, 0=success)",
			[]string{"collector"},
			nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (c *XiraidCollector) Describe(ch chan<- *prometheus.Desc) {
	// RAID basic metrics
	ch <- c.raidInfo

	ch <- c.raidConfig

	ch <- c.raidActive

	ch <- c.raidSizeBytes

	ch <- c.raidState

	ch <- c.raidMemoryUsageBytes

	ch <- c.raidDeviceState

	// RAID extended metrics
	ch <- c.raidCPUAllowed

	ch <- c.raidInitPrio

	ch <- c.raidReconPrio

	ch <- c.raidRestripePrio

	ch <- c.raidSDCPrio

	ch <- c.raidRequestLimit

	ch <- c.raidSchedEnabled

	ch <- c.raidMemoryLimitBytes

	ch <- c.raidMemoryPreallocBytes

	ch <- c.raidAdaptiveMerge

	ch <- c.raidMergeReadEnabled

	ch <- c.raidMergeReadMaxUsecs

	ch <- c.raidMergeReadWaitUsecs

	ch <- c.raidMergeWriteEnabled

	ch <- c.raidMergeWriteMaxUsecs

	ch <- c.raidMergeWriteWaitUsecs

	ch <- c.raidDeviceHealthPercent

	ch <- c.raidDeviceWearPercent

	// Drive metrics
	ch <- c.driveFaultySectorsTotal

	// License metrics
	ch <- c.licenseValid

	ch <- c.licenseExpiryTimestamp

	ch <- c.licenseDisksTotal

	ch <- c.licenseDisksInUse

	// Mail metrics
	ch <- c.mailConfigured

	// Pool metrics
	ch <- c.poolInfo

	// Settings metrics
	ch <- c.settingsAuthPort

	ch <- c.settingsClusterPoolAutoactivate

	ch <- c.settingsClusterRaidAutostart

	ch <- c.settingsEulaAccepted

	ch <- c.settingsFaultyCountThreshold

	ch <- c.settingsMailPollingInterval

	ch <- c.settingsMailProgressInterval

	ch <- c.settingsPoolReplaceDelay

	ch <- c.settingsScannerLoedEnabled

	ch <- c.settingsScannerPollingInterval

	ch <- c.settingsScannerSmartInterval

	// Scrape errors
	ch <- c.scrapeError
}

// Collect implements prometheus.Collector.
func (c *XiraidCollector) Collect(ch chan<- prometheus.Metric) {
	if c.config.EnableRaid {
		c.collectRaidMetrics(ch)
	}

	if c.config.EnableRaidExtended {
		c.collectRaidExtendedMetrics(ch)
	}

	if c.config.EnableDriveFaulty {
		c.collectDriveFaultyMetrics(ch)
	}

	if c.config.EnableLicense {
		c.collectLicenseMetrics(ch)
	}

	if c.config.EnableMail {
		c.collectMailMetrics(ch)
	}

	if c.config.EnablePool {
		c.collectPoolMetrics(ch)
	}

	if c.config.EnableSettings {
		c.collectSettingsMetrics(ch)
	}
}

func (c *XiraidCollector) collectRaidMetrics(ch chan<- prometheus.Metric) {
	raids, err := c.executor.GetRaidInfo()
	if err != nil {
		log.Printf("Error collecting RAID metrics: %v", err)

		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 1, "raid")

		return
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 0, "raid")

	for name, raid := range raids {
		// RAID info metric with static config as labels
		ch <- prometheus.MustNewConstMetric(
			c.raidInfo,
			prometheus.GaugeValue,
			1,
			name, raid.UUID, raid.Level,
			strconv.Itoa(raid.BlockSize),
			strconv.Itoa(raid.StripSize),
			strconv.Itoa(raid.GroupSize),
			raid.Sparepool,
		)

		// Config status
		config := 0.0
		if raid.Config {
			config = 1.0
		}

		ch <- prometheus.MustNewConstMetric(c.raidConfig, prometheus.GaugeValue, config, name, raid.UUID)

		// Active status
		active := 0.0
		if raid.Active {
			active = 1.0
		}

		ch <- prometheus.MustNewConstMetric(c.raidActive, prometheus.GaugeValue, active, name, raid.UUID)

		// Size in bytes
		sizeBytes := parseSizeToBytes(raid.Size)
		ch <- prometheus.MustNewConstMetric(c.raidSizeBytes, prometheus.GaugeValue, sizeBytes, name, raid.UUID)

		// State metrics
		c.collectRaidStates(ch, name, raid)

		// Memory usage (convert MB to bytes)
		memoryBytes := float64(raid.MemoryUsageMB) * 1024 * 1024
		ch <- prometheus.MustNewConstMetric(c.raidMemoryUsageBytes, prometheus.GaugeValue, memoryBytes, name, raid.UUID)

		// Device state metrics
		c.collectDeviceStates(ch, name, raid)
	}
}

func (c *XiraidCollector) collectRaidExtendedMetrics(ch chan<- prometheus.Metric) {
	raids, err := c.executor.GetRaidInfoExtended()
	if err != nil {
		log.Printf("Error collecting RAID extended metrics: %v", err)

		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 1, "raid_extended")

		return
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 0, "raid_extended")

	for name, raid := range raids {
		// CPU allowed count
		ch <- prometheus.MustNewConstMetric(
			c.raidCPUAllowed,
			prometheus.GaugeValue,
			float64(len(raid.CPUAllowed)),
			name, raid.UUID,
		)

		// Priority metrics
		if raid.InitPrio > 0 {
			ch <- prometheus.MustNewConstMetric(c.raidInitPrio, prometheus.GaugeValue, float64(raid.InitPrio), name, raid.UUID)
		}

		if raid.ReconPrio > 0 {
			ch <- prometheus.MustNewConstMetric(c.raidReconPrio, prometheus.GaugeValue, float64(raid.ReconPrio), name, raid.UUID)
		}

		if raid.RestripePrio > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.raidRestripePrio, prometheus.GaugeValue, float64(raid.RestripePrio), name, raid.UUID,
			)
		}

		if raid.SDCPrio > 0 {
			ch <- prometheus.MustNewConstMetric(c.raidSDCPrio, prometheus.GaugeValue, float64(raid.SDCPrio), name, raid.UUID)
		}

		// Request limit
		ch <- prometheus.MustNewConstMetric(
			c.raidRequestLimit, prometheus.GaugeValue, float64(raid.RequestLimit), name, raid.UUID,
		)

		// Scheduler enabled
		ch <- prometheus.MustNewConstMetric(
			c.raidSchedEnabled, prometheus.GaugeValue, float64(raid.SchedEnabled), name, raid.UUID,
		)

		// Memory metrics (convert MB to bytes)
		ch <- prometheus.MustNewConstMetric(
			c.raidMemoryLimitBytes,
			prometheus.GaugeValue,
			float64(raid.MemoryLimitMB)*1024*1024,
			name, raid.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			c.raidMemoryPreallocBytes,
			prometheus.GaugeValue,
			float64(raid.MemoryPreallocMB)*1024*1024,
			name, raid.UUID,
		)

		// Merge settings
		if raid.AdaptiveMerge >= 0 {
			ch <- prometheus.MustNewConstMetric(
				c.raidAdaptiveMerge, prometheus.GaugeValue, float64(raid.AdaptiveMerge), name, raid.UUID,
			)
		}

		if raid.MergeReadEnabled >= 0 {
			ch <- prometheus.MustNewConstMetric(
				c.raidMergeReadEnabled, prometheus.GaugeValue, float64(raid.MergeReadEnabled), name, raid.UUID,
			)
		}

		if raid.MergeWriteEnabled >= 0 {
			ch <- prometheus.MustNewConstMetric(
				c.raidMergeWriteEnabled, prometheus.GaugeValue, float64(raid.MergeWriteEnabled), name, raid.UUID,
			)
		}

		// Device health and wear
		c.collectDeviceHealthWear(ch, name, raid)
	}
}

func (c *XiraidCollector) collectDriveFaultyMetrics(ch chan<- prometheus.Metric) {
	// Get RAID info to map devices to RAID arrays
	raids, err := c.executor.GetRaidInfo()
	if err != nil {
		log.Printf("Error collecting RAID info for device mapping: %v", err)

		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 1, "drive_faulty")

		return
	}

	// Build device to RAID mapping
	c.buildDeviceToRaidMapping(raids)

	// Get faulty counts
	faultyCounts, err := c.executor.GetFaultyCount()
	if err != nil {
		log.Printf("Error collecting drive faulty counts: %v", err)

		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 1, "drive_faulty")

		return
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 0, "drive_faulty")

	for device, count := range faultyCounts {
		// Look up RAID info for this device
		raidInfo, found := c.deviceToRaidCache[device]
		if !found {
			// Device not in any RAID array, use empty labels
			raidInfo = deviceRaidInfo{raidName: "", uuid: ""}
		}

		ch <- prometheus.MustNewConstMetric(
			c.driveFaultySectorsTotal,
			prometheus.CounterValue,
			float64(count),
			raidInfo.raidName, raidInfo.uuid, device,
		)
	}
}

// buildDeviceToRaidMapping builds a mapping from device paths to RAID info.
func (c *XiraidCollector) buildDeviceToRaidMapping(raids map[string]xicli.RaidInfo) {
	// Clear the cache
	c.deviceToRaidCache = make(map[string]deviceRaidInfo)

	// Build the mapping
	for raidName, raid := range raids {
		for _, device := range raid.Devices {
			if len(device) < 2 {
				continue
			}

			devicePath, ok := device[1].(string)
			if !ok {
				continue
			}

			c.deviceToRaidCache[devicePath] = deviceRaidInfo{
				raidName: raidName,
				uuid:     raid.UUID,
			}
		}
	}
}

// collectRaidStates collects RAID state metrics.
func (c *XiraidCollector) collectRaidStates(ch chan<- prometheus.Metric, name string, raid xicli.RaidInfo) {
	possibleStates := []string{
		"online", "initialized", "initing", "inconsistent", "degraded",
		"reconstructing", "offline", "need_recon", "read_only", "restriping",
	}
	for _, possibleState := range possibleStates {
		value := 0.0

		for _, actualState := range raid.State {
			if actualState == possibleState {
				value = 1.0

				break
			}
		}

		ch <- prometheus.MustNewConstMetric(c.raidState, prometheus.GaugeValue, value, name, raid.UUID, possibleState)
	}
}

// collectDeviceStates collects device state metrics for a RAID array.
func (c *XiraidCollector) collectDeviceStates(
	ch chan<- prometheus.Metric, name string, raid xicli.RaidInfo,
) {
	for deviceIndex, device := range raid.Devices {
		c.collectDeviceState(ch, name, raid, deviceIndex, device)
	}
}

// collectDeviceState collects state metric for a single device.
func (c *XiraidCollector) collectDeviceState(
	ch chan<- prometheus.Metric, name string, raid xicli.RaidInfo, deviceIndex int, device []interface{},
) {
	if len(device) < 3 {
		return
	}

	devicePath, ok := device[1].(string)
	if !ok {
		return
	}

	serial := ""
	if deviceIndex < len(raid.Serials) {
		serial = raid.Serials[deviceIndex]
	}

	// Get device states
	deviceStates := extractDeviceStates(device)

	// Create state string
	stateStr := strings.Join(deviceStates, ",")
	if stateStr == "" {
		stateStr = "none"
	}

	// Enumerate state value
	stateValue := getDeviceStateValue(deviceStates)

	ch <- prometheus.MustNewConstMetric(
		c.raidDeviceState,
		prometheus.GaugeValue,
		stateValue,
		name, raid.UUID, devicePath, serial, stateStr,
	)
}

// extractDeviceStates extracts state strings from device data.
func extractDeviceStates(device []interface{}) []string {
	deviceStates := []string{}

	if states, ok := device[2].([]interface{}); ok {
		for _, state := range states {
			if stateStr, ok := state.(string); ok {
				deviceStates = append(deviceStates, stateStr)
			}
		}
	}

	return deviceStates
}

// getDeviceStateValue returns numeric value for device state (1=online, 2=offline, 0=none/other).
func getDeviceStateValue(deviceStates []string) float64 {
	for _, deviceState := range deviceStates {
		switch deviceState {
		case "online":
			return 1.0
		case "offline":
			return 2.0
		}
	}

	return 0.0
}

// collectDeviceHealthWear collects device health and wear metrics for a RAID array.
func (c *XiraidCollector) collectDeviceHealthWear(
	ch chan<- prometheus.Metric, name string, raid xicli.RaidInfo,
) {
	for deviceIndex, device := range raid.Devices {
		if len(device) < 3 {
			continue
		}

		devicePath, ok := device[1].(string)
		if !ok {
			continue
		}

		serial := ""
		if deviceIndex < len(raid.Serials) {
			serial = raid.Serials[deviceIndex]
		}

		// Device health
		if deviceIndex < len(raid.DevicesHealth) && raid.DevicesHealth[deviceIndex] != "" {
			health := parsePercentage(raid.DevicesHealth[deviceIndex])
			ch <- prometheus.MustNewConstMetric(
				c.raidDeviceHealthPercent,
				prometheus.GaugeValue,
				health,
				name, raid.UUID, devicePath, serial,
			)
		}

		// Device wear
		if deviceIndex < len(raid.DevicesWear) && raid.DevicesWear[deviceIndex] != "" {
			wear := parsePercentage(raid.DevicesWear[deviceIndex])
			ch <- prometheus.MustNewConstMetric(
				c.raidDeviceWearPercent,
				prometheus.GaugeValue,
				wear,
				name, raid.UUID, devicePath, serial,
			)
		}
	}
}

func (c *XiraidCollector) collectLicenseMetrics(ch chan<- prometheus.Metric) {
	license, err := c.executor.GetLicenseInfo()
	if err != nil {
		log.Printf("Error collecting license info: %v", err)

		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 1, "license")

		return
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 0, "license")

	// License valid status
	valid := 0.0
	if license.Status == "valid" {
		valid = 1.0
	}

	ch <- prometheus.MustNewConstMetric(c.licenseValid, prometheus.GaugeValue, valid, license.Status, license.Type)

	// License expiry timestamp
	expiryTimestamp := parseDateToTimestamp(license.Expired)
	ch <- prometheus.MustNewConstMetric(c.licenseExpiryTimestamp, prometheus.GaugeValue, expiryTimestamp, license.Type)

	// Disk counts
	disksTotal, _ := strconv.ParseFloat(license.Disks, 64)
	disksInUse, _ := strconv.ParseFloat(license.DisksInUse, 64)

	ch <- prometheus.MustNewConstMetric(c.licenseDisksTotal, prometheus.GaugeValue, disksTotal, license.Type)

	ch <- prometheus.MustNewConstMetric(c.licenseDisksInUse, prometheus.GaugeValue, disksInUse, license.Type)
}

func (c *XiraidCollector) collectMailMetrics(ch chan<- prometheus.Metric) {
	mail, err := c.executor.GetMailInfo()
	if err != nil {
		log.Printf("Error collecting mail info: %v", err)

		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 1, "mail")

		return
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 0, "mail")

	// Mail configured status
	configured := 0.0
	if mail.Recipient != "" && mail.Level != "" {
		configured = 1.0
	}

	ch <- prometheus.MustNewConstMetric(c.mailConfigured, prometheus.GaugeValue, configured, mail.Recipient, mail.Level)
}

func (c *XiraidCollector) collectPoolMetrics(ch chan<- prometheus.Metric) {
	pool, err := c.executor.GetPoolInfo()
	if err != nil {
		log.Printf("Error collecting pool info: %v", err)

		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 1, "pool")

		return
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 0, "pool")

	// Pool info metric
	value := 0.0
	if pool.Name != "" {
		value = 1.0
	}

	ch <- prometheus.MustNewConstMetric(
		c.poolInfo,
		prometheus.GaugeValue,
		value,
		pool.Name, pool.Devices, pool.Serials, pool.Sizes, pool.State,
	)
}

func (c *XiraidCollector) collectSettingsMetrics(ch chan<- prometheus.Metric) {
	// Collect all settings
	auth, errAuth := c.executor.GetSettingsAuth()
	cluster, errCluster := c.executor.GetSettingsCluster()
	eula, errEula := c.executor.GetSettingsEula()
	faultyCount, errFaultyCount := c.executor.GetSettingsFaultyCount()
	mail, errMail := c.executor.GetSettingsMail()
	pool, errPool := c.executor.GetSettingsPool()
	scanner, errScanner := c.executor.GetSettingsScanner()

	// Log and track errors
	hasError := c.logSettingsErrors(errAuth, errCluster, errEula, errFaultyCount, errMail, errPool, errScanner)

	// Report scrape status
	if hasError {
		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 1, "settings")
	} else {
		ch <- prometheus.MustNewConstMetric(c.scrapeError, prometheus.GaugeValue, 0, "settings")
	}

	// Collect individual settings metrics
	c.collectAuthSettings(ch, auth, errAuth)
	c.collectClusterSettings(ch, cluster, errCluster)
	c.collectEulaSettings(ch, eula, errEula)
	c.collectFaultyCountSettings(ch, faultyCount, errFaultyCount)
	c.collectMailSettings(ch, mail, errMail)
	c.collectPoolSettings(ch, pool, errPool)
	c.collectScannerSettings(ch, scanner, errScanner)
}

// logSettingsErrors logs errors from settings collection and returns whether any errors occurred.
func (c *XiraidCollector) logSettingsErrors(
	errAuth, errCluster, errEula, errFaultyCount, errMail, errPool, errScanner error,
) bool {
	hasError := false

	if errAuth != nil {
		log.Printf("Error collecting settings auth: %v", errAuth)

		hasError = true
	}

	if errCluster != nil {
		log.Printf("Error collecting settings cluster: %v", errCluster)

		hasError = true
	}

	if errEula != nil {
		log.Printf("Error collecting settings eula: %v", errEula)

		hasError = true
	}

	if errFaultyCount != nil {
		log.Printf("Error collecting settings faulty-count: %v", errFaultyCount)

		hasError = true
	}

	if errMail != nil {
		log.Printf("Error collecting settings mail: %v", errMail)

		hasError = true
	}

	if errPool != nil {
		log.Printf("Error collecting settings pool: %v", errPool)

		hasError = true
	}

	if errScanner != nil {
		log.Printf("Error collecting settings scanner: %v", errScanner)

		hasError = true
	}

	return hasError
}

func (c *XiraidCollector) collectAuthSettings(
	ch chan<- prometheus.Metric, auth *xicli.SettingsAuth, err error,
) {
	if err == nil {
		ch <- prometheus.MustNewConstMetric(c.settingsAuthPort, prometheus.GaugeValue, float64(auth.Port), auth.Host)
	}
}

func (c *XiraidCollector) collectClusterSettings(
	ch chan<- prometheus.Metric, cluster *xicli.SettingsCluster, err error,
) {
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			c.settingsClusterPoolAutoactivate, prometheus.GaugeValue, float64(cluster.PoolAutoactivate),
		)

		ch <- prometheus.MustNewConstMetric(
			c.settingsClusterRaidAutostart, prometheus.GaugeValue, float64(cluster.RaidAutostart),
		)
	}
}

func (c *XiraidCollector) collectEulaSettings(
	ch chan<- prometheus.Metric, eula *xicli.SettingsEula, err error,
) {
	if err == nil {
		eulaValue := 0.0
		if eula.Status == "accepted" {
			eulaValue = 1.0
		}

		ch <- prometheus.MustNewConstMetric(c.settingsEulaAccepted, prometheus.GaugeValue, eulaValue, eula.Status)
	}
}

func (c *XiraidCollector) collectFaultyCountSettings(
	ch chan<- prometheus.Metric, faultyCount *xicli.SettingsFaultyCount, err error,
) {
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			c.settingsFaultyCountThreshold, prometheus.GaugeValue, float64(faultyCount.FaultyCountThreshold),
		)
	}
}

func (c *XiraidCollector) collectMailSettings(
	ch chan<- prometheus.Metric, mail *xicli.SettingsMail, err error,
) {
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			c.settingsMailPollingInterval, prometheus.GaugeValue, float64(mail.PollingInterval),
		)

		ch <- prometheus.MustNewConstMetric(
			c.settingsMailProgressInterval, prometheus.GaugeValue, float64(mail.ProgressPollingInterval),
		)
	}
}

func (c *XiraidCollector) collectPoolSettings(
	ch chan<- prometheus.Metric, pool *xicli.SettingsPool, err error,
) {
	if err == nil {
		ch <- prometheus.MustNewConstMetric(c.settingsPoolReplaceDelay, prometheus.GaugeValue, float64(pool.ReplaceDelay))
	}
}

func (c *XiraidCollector) collectScannerSettings(
	ch chan<- prometheus.Metric, scanner *xicli.SettingsScanner, err error,
) {
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			c.settingsScannerLoedEnabled, prometheus.GaugeValue, float64(scanner.LoedEnabled),
		)

		ch <- prometheus.MustNewConstMetric(
			c.settingsScannerPollingInterval, prometheus.GaugeValue, float64(scanner.ScannerPollingInterval),
		)

		ch <- prometheus.MustNewConstMetric(
			c.settingsScannerSmartInterval, prometheus.GaugeValue, float64(scanner.SmartPollingInterval),
		)
	}
}

// parsePercentage converts "100%" to 100.0.
func parsePercentage(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	return val
}

// parseSizeToBytes converts size strings like "1788 GiB" or "128767 GiB" to bytes.
func parseSizeToBytes(s string) float64 {
	s = strings.TrimSpace(s)

	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(parts[1])
	switch unit {
	case "B":
		return value
	case "KB", "KIB":
		return value * 1024
	case "MB", "MIB":
		return value * 1024 * 1024
	case "GB", "GIB":
		return value * 1024 * 1024 * 1024
	case "TB", "TIB":
		return value * 1024 * 1024 * 1024 * 1024
	default:
		return 0
	}
}

// parseDateToTimestamp converts date strings like "2125-7-31" to Unix timestamp.
func parseDateToTimestamp(s string) float64 {
	// Try parsing as YYYY-M-D or YYYY-MM-DD
	layouts := []string{"2006-1-2", "2006-01-02"}
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return float64(t.Unix())
		}
	}

	return 0
}
