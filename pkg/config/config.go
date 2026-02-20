package config

import (
	"flag"
	"strings"
	"time"
)

const DefaultVessels = "OTSO,KONTIO,POLARIS,URHO,SISU,VOIMA,FENNICA,NORDICA,ALE,ATLE,FREJ,ODEN,YMER,IDUN,KRONPRINS HAAKON,SVALBARD"

type Config struct {
	ListenAddress   string
	MetricsPath     string
	VesselsURL      string
	LocationsURL    string
	DigitrafficUser string
	RefreshInterval time.Duration
	RequestTimeout  time.Duration
	TargetNames     map[string]struct{}
}

func ParseFlags() (Config, error) {
	listenAddress := flag.String("listen-address", ":9877", "Address the exporter listens on")
	metricsPath := flag.String("metrics-path", "/metrics", "Path to expose Prometheus metrics")
	vesselsURL := flag.String("vessels-url", "https://meri.digitraffic.fi/api/ais/v1/vessels", "Digitraffic AIS vessels endpoint")
	locationsURL := flag.String("locations-url", "https://meri.digitraffic.fi/api/ais/v1/locations", "Digitraffic AIS locations endpoint")
	digitrafficUser := flag.String("digitraffic-user", "icebreaker-exporter/1.0", "Value for the Digitraffic-User request header")
	refreshInterval := flag.Duration("refresh-interval", 2*time.Minute, "How often to refresh vessel positions")
	requestTimeout := flag.Duration("request-timeout", 20*time.Second, "Timeout for each Digitraffic request")
	targetVessels := flag.String("vessel-names", DefaultVessels, "Comma separated list of vessel names to export")
	flag.Parse()

	cfg := Config{
		ListenAddress:   *listenAddress,
		MetricsPath:     *metricsPath,
		VesselsURL:      *vesselsURL,
		LocationsURL:    *locationsURL,
		DigitrafficUser: *digitrafficUser,
		RefreshInterval: *refreshInterval,
		RequestTimeout:  *requestTimeout,
		TargetNames:     ParseTargetNames(*targetVessels),
	}
	return cfg, nil
}

func ParseTargetNames(value string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, item := range strings.Split(value, ",") {
		norm := NormalizeName(item)
		if norm == "" {
			continue
		}
		out[norm] = struct{}{}
	}
	return out
}

func NormalizeName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}
