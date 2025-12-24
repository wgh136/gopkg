# Server Status

A Go package for collecting server status metrics (CPU, RAM, Swap, Network traffic, Disk IOPS) and reporting them to Prometheus.

## Features

- **CPU Usage**: Monitors CPU utilization percentage
- **Memory Usage**: Tracks RAM usage percentage
- **Swap Usage**: Monitors swap memory usage percentage
- **Network Traffic**: Measures network upload and download rates (bytes/second)
- **Disk IOPS**: Tracks disk read and write operations per second

## Installation

```bash
go get github.com/wgh136/gopkg/server_status
```

## Usage

This package provides a single function `Register()` to register the metrics collector with Prometheus:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/prometheus/client_golang/prometheus/promhttp"
    serverstatus "github.com/wgh136/gopkg/server_status"
)

func main() {
    // Register the server status collector
    if err := serverstatus.Register(nil); err != nil {
        log.Fatal(err)
    }
    
    // Expose metrics endpoint
    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Using a Custom Registry

You can also use a custom Prometheus registry:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    serverstatus "github.com/wgh136/gopkg/server_status"
)

func main() {
    // Create a custom registry
    registry := prometheus.NewRegistry()
    
    // Register the server status collector with custom registry
    if err := serverstatus.Register(registry); err != nil {
        log.Fatal(err)
    }
    
    // Expose metrics endpoint with custom registry
    http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Metrics Exposed

The package exposes the following Prometheus metrics:

| Metric Name | Type | Description |
|------------|------|-------------|
| `server_cpu_usage_percent` | Gauge | CPU usage percentage (0-100) |
| `server_memory_usage_percent` | Gauge | Memory usage percentage (0-100) |
| `server_swap_usage_percent` | Gauge | Swap usage percentage (0-100) |
| `server_network_receive_bytes_per_second` | Gauge | Network download rate in bytes/second |
| `server_network_transmit_bytes_per_second` | Gauge | Network upload rate in bytes/second |
| `server_disk_read_ops_per_second` | Gauge | Disk read operations per second |
| `server_disk_write_ops_per_second` | Gauge | Disk write operations per second |

## Example Prometheus Queries

```promql
# CPU usage
server_cpu_usage_percent

# Memory usage
server_memory_usage_percent

# Network total traffic (bytes/sec)
server_network_receive_bytes_per_second + server_network_transmit_bytes_per_second

# Total disk IOPS
server_disk_read_ops_per_second + server_disk_write_ops_per_second
```