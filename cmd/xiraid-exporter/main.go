// Package main is the entry point for the Xiraid Exporter
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"

	"github.com/E4-Computer-Engineering/xiraid-exporter/pkg/collector"
)

// Collector represents a metric collector with enable/disable capability.
type Collector struct {
	name         string
	defaultState bool
	enabled      *bool
	disabled     *bool
	description  string
}

var (
	collectors = map[string]*Collector{
		"raid": {
			name:         "raid",
			defaultState: true,
			description:  "RAID array basic metrics (info, config, state, size, memory usage, device states)",
		},
		"raid-extended": {
			name:         "raid-extended",
			defaultState: false,
			description:  "RAID array extended metrics (CPU, priorities, memory limits, merge settings, device health/wear)",
		},
		"drive-faulty": {
			name:         "drive-faulty",
			defaultState: true,
			description:  "Drive faulty sector counts",
		},
		"license": {
			name:         "license",
			defaultState: true,
			description:  "License validity, expiration, and disk usage",
		},
		"mail": {
			name:         "mail",
			defaultState: false,
			description:  "Mail notification configuration",
		},
		"pool": {
			name:         "pool",
			defaultState: false,
			description:  "Spare pool information",
		},
		"settings": {
			name:         "settings",
			defaultState: false,
			description:  "System settings (auth, cluster, EULA, thresholds, intervals)",
		},
	}

	disableDefaults = flag.Bool("collector.disable-defaults", false, "Disable all default collectors")
	listenAddress   = flag.String("web.listen-address", ":9827", "Address on which to expose metrics and web interface")
	metricsPath     = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
	xicliPath       = flag.String("xicli.path", "xicli", "Path to the xicli binary")
)

func initCollectorFlags() {
	// Register flags for each collector
	for name, collector := range collectors {
		enableFlagName := "collector." + name
		disableFlagName := "no-collector." + name

		// --collector.X flag to enable
		enableFlag := flag.Bool(
			enableFlagName,
			collector.defaultState,
			fmt.Sprintf("Enable the %s collector (default: %t)", collector.description, collector.defaultState),
		)

		// --no-collector.X flag to disable
		disableFlag := flag.Bool(
			disableFlagName,
			false,
			fmt.Sprintf("Disable the %s collector", collector.description),
		)

		collector.enabled = enableFlag
		collector.disabled = disableFlag
		collectors[name] = collector
	}
}

func resolveCollectorStates() map[string]bool {
	states := make(map[string]bool)

	for name, collector := range collectors {
		// Start with default state
		enabled := collector.defaultState

		// If disable-defaults is set, start with false
		if *disableDefaults {
			enabled = false
		}

		// Check if flags were explicitly set
		explicitlyEnabled := false
		explicitlyDisabled := false

		flag.Visit(func(visitedFlag *flag.Flag) {
			if visitedFlag.Name == "collector."+name {
				explicitlyEnabled = true
				enabled = *collector.enabled
			}

			if visitedFlag.Name == "no-collector."+name {
				explicitlyDisabled = true
			}
		})

		// Disable flag takes precedence
		if explicitlyDisabled {
			enabled = false
		} else if explicitlyEnabled {
			enabled = true
		}

		states[name] = enabled
	}

	return states
}

func printUsage() {
	fmt.Println("xiraid_exporter - Prometheus exporter for Xiraid RAID systems")
	fmt.Println("\nExports Xiraid RAID array metrics in Prometheus format using the xicli command-line tool.")
	fmt.Println("\nUsage: xiraid_exporter [options]")
	fmt.Println("\nWeb server options:")
	fmt.Println("  --web.listen-address string")
	fmt.Println("        Address on which to expose metrics and web interface (default \":9827\")")
	fmt.Println("  --web.telemetry-path string")
	fmt.Println("        Path under which to expose metrics (default \"/metrics\")")
	fmt.Println("\nXicli options:")
	fmt.Println("  --xicli.path string")
	fmt.Println("        Path to the xicli binary (default \"xicli\")")
	fmt.Println("\nCollector options:")
	fmt.Println("  --collector.<name>")
	fmt.Println("        Enable the specified collector")
	fmt.Println("  --no-collector.<name>")
	fmt.Println("        Disable the specified collector")
	fmt.Println("  --collector.disable-defaults")
	fmt.Println("        Disable all default collectors")
	fmt.Println("\nAvailable collectors:")

	for name, collector := range collectors {
		defaultStr := ""
		if collector.defaultState {
			defaultStr = " (enabled by default)"
		}

		fmt.Printf("  %-15s %s%s\n", name, collector.description, defaultStr)
	}

	fmt.Println("\nExamples:")
	fmt.Println("  # Start with all default collectors (raid, drive-faulty, license)")
	fmt.Println("  xiraid_exporter")
	fmt.Println("\n  # Enable extended RAID metrics in addition to defaults")
	fmt.Println("  xiraid_exporter --collector.raid-extended")
	fmt.Println("\n  # Disable license collector")
	fmt.Println("  xiraid_exporter --no-collector.license")
	fmt.Println("\n  # Enable only RAID basic and extended metrics")
	fmt.Println("  xiraid_exporter --collector.disable-defaults --collector.raid --collector.raid-extended")
	fmt.Println("\n  # Enable all collectors on custom port")
	fmt.Println("  xiraid_exporter --web.listen-address=\":9100\" --collector.raid-extended \\")
	fmt.Println("    --collector.mail --collector.pool --collector.settings")
	fmt.Println("\n  # Use custom xicli path")
	fmt.Println("  xiraid_exporter --xicli.path=\"/usr/local/bin/xicli\"")
}

func main() {
	// Initialize collector flags before parsing
	initCollectorFlags()

	flag.Usage = printUsage

	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Resolve collector states based on flags
	collectorStates := resolveCollectorStates()

	// Log enabled collectors
	log.Printf("Enabled collectors:")

	for name, enabled := range collectorStates {
		if enabled {
			log.Printf("  - %s", name)
		}
	}

	xiraidCollector := collector.NewXiraidCollector(*xicliPath, collector.Config{
		EnableRaid:         collectorStates["raid"],
		EnableRaidExtended: collectorStates["raid-extended"],
		EnableDriveFaulty:  collectorStates["drive-faulty"],
		EnableLicense:      collectorStates["license"],
		EnableMail:         collectorStates["mail"],
		EnablePool:         collectorStates["pool"],
		EnableSettings:     collectorStates["settings"],
	})
	prometheus.MustRegister(xiraidCollector)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			http.NotFound(writer, req)

			return
		}

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(writer, `<html>
<head><title>Xiraid Exporter</title></head>
<body>
<h1>Xiraid Exporter</h1>
<p><a href="%s">Metrics</a></p>
</body>
</html>`, *metricsPath)
	})

	log.Printf("Starting xiraid_exporter on %s", *listenAddress)
	log.Printf("Metrics path: %s", *metricsPath)

	// #nosec G114 - This is a metrics server with intentionally no timeouts
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
