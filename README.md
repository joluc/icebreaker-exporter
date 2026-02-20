# Icebreaker Prometheus Exporter

![Go Version](https://img.shields.io/github/go-mod/go-version/joluc/icebreaker-exporter?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)

A lightweight Prometheus exporter written in Go that fetches live vessel coordinates and metadata from the **Digitraffic AIS API** and exposes them as Prometheus metrics. Track the real-time positions of the Nordic icebreaker fleet!

---

## Exported Metrics

| Metric Name | Type | Description |
|---|---|---|
| `icebreaker_latitude_degrees` | Gauge | Current latitude of a Nordic icebreaker |
| `icebreaker_longitude_degrees` | Gauge | Current longitude of a Nordic icebreaker |
| `icebreaker_last_report_timestamp_seconds`| Gauge | Unix timestamp of when the ship last reported |
| `icebreaker_report_age_seconds` | Gauge | Age of the last report in seconds |
| `icebreaker_up` | Gauge | `1` if the Digitraffic API was successfully refreshed |
| `icebreaker_last_refresh_timestamp_seconds` | Gauge | Unix timestamp of the last refresh |
| `icebreaker_refresh_duration_seconds` | Gauge | Duration of the latest Digitraffic fetch operation |
| `icebreaker_scrapes_total` | Counter | Total number of HTTP `/metrics` scrapes |
| `icebreaker_positions` | Gauge | Number of valid icebreaker positions currently being tracked |

## Getting Started

### Run Locally

The easiest way to test the exporter is to run it natively using Go:

```bash
go run ./cmd/icebreaker-exporter \
  -listen-address :9877 \
  -metrics-path /metrics \
  -digitraffic-user "my-team/icebreaker-exporter-exporter" \
  -refresh-interval 2m
```

Then scrape the endpoint:

```bash
curl http://localhost:9877/metrics
```

### Configuration Flags

You can customize the exporter runtime using the following command-line flags:

| Flag | Default | Description |
|---|---|---|
| `-listen-address` | `:9877` | Address to listen on for HTTP requests. |
| `-metrics-path` | `/metrics` | Path under which to expose metrics. |
| `-vessels-url` | `https://meri...` | URL for the Digitraffic Vessels API. |
| `-locations-url` | `https://meri...` | URL for the Digitraffic Locations API. |
| `-digitraffic-user`| `icebreake..`| **Required.** A descriptive identifier (like a User-Agent) sent to the Digitraffic API (e.g., `your-name/app-name`). No API key or registration is needed. |
| `-refresh-interval`| `2m` | Interval between Digitraffic API refreshes. |
| `-request-timeout` | `20s` | HTTP timeout for Digitraffic requests. |
| `-vessel-names` | *See below* | Comma-separated list of icebreaker names. |

**Default Monitored Nordic Icebreakers:**
- **FI**: `OTSO`, `KONTIO`, `POLARIS`, `URHO`, `SISU`, `VOIMA`, `FENNICA`, `NORDICA`
- **SE**: `ALE`, `ATLE`, `FREJ`, `ODEN`, `YMER`, `IDUN`
- **NO**: `KRONPRINS HAAKON`, `SVALBARD`

*(Note: Denmark decommissioned their state icebreakers in 2012, and neither Iceland nor Greenland operate dedicated state icebreakers. Therefore, no active DK/IS/GL ships are included in the defaults.)*

## Examples

### Example Metrics Output

When you curl the `/metrics` endpoint, you will see output similar to this:

```text
# HELP icebreaker_up Whether the latest Digitraffic refresh succeeded
# TYPE icebreaker_up gauge
icebreaker_up 1
# HELP icebreaker_last_refresh_timestamp_seconds Unix timestamp of last refresh
# TYPE icebreaker_last_refresh_timestamp_seconds gauge
icebreaker_last_refresh_timestamp_seconds 1708453488
# HELP icebreaker_refresh_duration_seconds Duration of latest refresh operation
# TYPE icebreaker_refresh_duration_seconds gauge
icebreaker_refresh_duration_seconds 0.354123
# HELP icebreaker_positions Number of exported icebreaker positions
# TYPE icebreaker_positions gauge
icebreaker_positions 3
# HELP icebreaker_latitude_degrees Current latitude of a Nordic icebreaker
# TYPE icebreaker_latitude_degrees gauge
icebreaker_latitude_degrees{vessel_name="OTSO",mmsi="230124000"} 65.123456
# HELP icebreaker_longitude_degrees Current longitude of a Nordic icebreaker
# TYPE icebreaker_longitude_degrees gauge
icebreaker_longitude_degrees{vessel_name="OTSO",mmsi="230124000"} 24.987654
```
