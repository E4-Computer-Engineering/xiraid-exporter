# Xiraid Exporter

[![build](https://github.com/E4-Computer-Engineering/xiraid-exporter/actions/workflows/build.yml/badge.svg)](https://github.com/E4-Computer-Engineering/xiraid-exporter/actions/workflows/build.yml)
![Latest GitHub release](https://img.shields.io/github/release/E4-Computer-Engineering/xiraid-exporter.svg)
[![GitHub license](https://img.shields.io/github/license/E4-Computer-Engineering/xiraid-exporter)](https://github.com/E4-Computer-Engineering/xiraid-exporter/blob/master/LICENSE)
![GitHub all releases](https://img.shields.io/github/downloads/E4-Computer-Engineering/xiraid-exporter/total)

A Prometheus exporter for Xiraid RAID systems. This exporter collects metrics from Xiraid arrays using the `xicli` command-line tool and exposes them in Prometheus format.

## Features

- Modular collector system with enable/disable flags
- Monitors RAID array health and status
- Tracks device health and wear levels
- Monitors memory usage per array
- Tracks faulty sector counts per drive
- Monitors license status and expiration
- Collects mail, pool, and settings configuration
- Exposes metrics in Prometheus format

## Collectors

The exporter provides multiple collectors that can be individually enabled or disabled:

### RAID Collector (default: enabled)

Collects basic RAID array information from `xicli raid show`:

- RAID configuration and status
- Array states (online, initialized, etc.)
- Memory usage
- Device states

### RAID Extended Collector (default: disabled)

Collects extended RAID metrics from `xicli raid show --extended`:

- CPU allocation
- Priority settings (init, recon, restripe, SDC)
- Memory limits and preallocation
- Merge settings
- Device health and wear percentages

### Drive Faulty Count Collector (default: enabled)

Collects faulty sector counts per drive from `xicli drive faulty-count show`.

### License Collector (default: enabled)

Collects license information from `xicli license show`:

- License validity and expiration
- Disk count and usage

### Mail Collector (default: disabled)

Collects mail notification configuration from `xicli mail show`.

### Pool Collector (default: disabled)

Collects spare pool information from `xicli pool show`.

### Settings Collector (default: disabled)

Collects various system settings from multiple `xicli settings` commands:

- Authentication settings
- Cluster settings
- EULA status
- Faulty count thresholds
- Mail polling intervals
- Pool replace delays
- Scanner settings

## Metrics

### RAID Basic Metrics (RAID Collector)

- `xiraid_raid_info` - RAID array information with static parameters as labels
- `xiraid_raid_config` - RAID array in configuration (1=yes, 0=no)
- `xiraid_raid_active` - RAID array block device is active (1=active, 0=inactive)
- `xiraid_raid_size_bytes` - RAID array total size in bytes
- `xiraid_raid_state` - RAID array state (enumerated for all possible states)
- `xiraid_raid_memory_usage_bytes` - RAID array memory usage in bytes
- `xiraid_raid_device_state` - RAID device state (1=online, 2=offline, 0=none)

### RAID Extended Metrics (RAID Extended Collector)

- `xiraid_raid_cpu_allowed_count` - Number of CPUs allowed
- `xiraid_raid_init_priority` - Initialization priority
- `xiraid_raid_recon_priority` - Reconstruction priority
- `xiraid_raid_restripe_priority` - Restripe priority
- `xiraid_raid_sdc_priority` - Silent Data Corruption scan priority
- `xiraid_raid_request_limit` - Request limit
- `xiraid_raid_scheduler_enabled` - Scheduler enabled
- `xiraid_raid_memory_limit_bytes` - Memory limit in bytes
- `xiraid_raid_memory_prealloc_bytes` - Preallocated memory in bytes
- `xiraid_raid_adaptive_merge` - Adaptive merge enabled
- `xiraid_raid_merge_read_enabled` - Merge read enabled
- `xiraid_raid_merge_read_max_microseconds` - Merge read max microseconds
- `xiraid_raid_merge_read_wait_microseconds` - Merge read wait microseconds
- `xiraid_raid_merge_write_enabled` - Merge write enabled
- `xiraid_raid_merge_write_max_microseconds` - Merge write max microseconds
- `xiraid_raid_merge_write_wait_microseconds` - Merge write wait microseconds
- `xiraid_device_health_percent` - RAID device health percentage
- `xiraid_device_wear_percent` - RAID device wear percentage

### Drive Metrics (Drive Faulty Count Collector)

- `xiraid_drive_faulty_sectors_total` - Total number of faulty sectors on drive

### License Metrics (License Collector)

- `xiraid_license_valid` - License is valid (1=valid, 0=invalid)
- `xiraid_license_expiry_timestamp_seconds` - License expiry timestamp in seconds since epoch
- `xiraid_license_disks_total` - Total number of disks allowed by license
- `xiraid_license_disks_in_use` - Number of disks currently in use

### Mail Metrics (Mail Collector)

- `xiraid_mail_configured` - Mail notification is configured

### Pool Metrics (Pool Collector)

- `xiraid_pool_info` - Pool information with labels

### Settings Metrics (Settings Collector)

- `xiraid_settings_auth_port` - Authentication service port
- `xiraid_settings_cluster_pool_autoactivate` - Pool autoactivate setting
- `xiraid_settings_cluster_raid_autostart` - RAID autostart setting
- `xiraid_settings_eula_accepted` - EULA acceptance status
- `xiraid_settings_faulty_count_threshold` - Faulty sector count threshold
- `xiraid_settings_mail_polling_interval_seconds` - Mail polling interval
- `xiraid_settings_mail_progress_polling_interval_seconds` - Mail progress polling interval
- `xiraid_settings_pool_replace_delay_seconds` - Pool drive replace delay
- `xiraid_settings_scanner_loed_enabled` - Scanner LOED enabled
- `xiraid_settings_scanner_polling_interval_seconds` - Scanner polling interval
- `xiraid_settings_scanner_smart_interval_seconds` - Scanner SMART polling interval

### Scrape Error Metrics

- `xiraid_scrape_error` - Whether an error occurred during the last scrape (per collector)

## Requirements

- Go 1.23 or later
- Xiraid system with `xicli` command-line tool installed
- Access to execute `xicli` commands

## Installation

### From Source

```bash
git clone https://github.com/E4-Computer-Engineering/xiraid-exporter.git
cd xiraid-exporter
go build -o xiraid-exporter .
```

### Using GoReleaser

```bash
goreleaser release --snapshot --skip=publish --clean
```

## Usage

```bash
./xiraid-exporter [flags]
```

### Flags

#### Server Flags

- `--web.listen-address` - Address to listen on for web interface and telemetry (default: `:9827`)
- `--web.telemetry-path` - Path under which to expose metrics (default: `/metrics`)
- `--xicli.path` - Path to the xicli binary (default: `xicli`)

#### Collector Flags

Following the [node_exporter](https://github.com/prometheus/node_exporter) pattern:

**Enabling collectors:**

- `--collector.<name>` - Enable a specific collector

**Disabling collectors:**

- `--no-collector.<name>` - Disable a specific collector (useful for default collectors)

**Disable all defaults:**

- `--collector.disable-defaults` - Disable all default collectors, then use `--collector.<name>` to enable only specific ones

**Available Collectors:**

- `raid` - RAID collector (default: **enabled**)
- `raid-extended` - RAID extended metrics collector (default: disabled)
- `drive-faulty` - Drive faulty count collector (default: **enabled**)
- `license` - License collector (default: **enabled**)
- `mail` - Mail collector (default: disabled)
- `pool` - Pool collector (default: disabled)
- `settings` - Settings collector (default: disabled)

### Examples

```bash
# Start the exporter with default collectors (raid, drive-faulty, license)
./xiraid-exporter

# Enable an additional collector (extended RAID metrics)
./xiraid-exporter --collector.raid-extended

# Disable a default collector
./xiraid-exporter --no-collector.license

# Enable only specific collectors (disable all defaults, then enable only what you want)
./xiraid-exporter --collector.disable-defaults --collector.raid --collector.raid-extended

# Complex example: all settings collectors but no license
./xiraid-exporter --no-collector.license --collector.mail --collector.pool --collector.settings

# Start on a custom port with extended RAID metrics
./xiraid-exporter --web.listen-address=":9999" --collector.raid-extended

# Use a custom path for xicli
./xiraid-exporter --xicli.path="/usr/local/bin/xicli"
```

## Prometheus Configuration

Add the following to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'xiraid'
    static_configs:
      - targets: ['localhost:9827']
```

## Development

### Building

```bash
go build -o xiraid-exporter .
```

### Linting

```bash
golangci-lint run --timeout=3m0s
```

### Testing

```bash
go test ./...
```

### CI/CD

This project uses GitHub Actions for CI/CD:

- Build workflow runs on push/PR to main branch
- Release workflow runs on version tags (v*.*.*)

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
