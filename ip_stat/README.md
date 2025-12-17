# IP Stat

A lightweight, thread-safe Go library for counting unique IP addresses within a sliding time window.

## Features

- **Sliding Window**: Tracks unique IPs over a configurable time window
- **Thread-Safe**: Safe for concurrent use across multiple goroutines
- **Memory Efficient**: Uses a bucket-based approach to minimize memory overhead
- **Non-Blocking**: IP additions are processed asynchronously via channels

## Installation

```bash
go get github.com/wgh136/gopkg/ip_stat
```

## Usage

```go
package main

import (
    "fmt"
    "time"
    
    ipstat "github.com/wgh136/gopkg/ip_stat"
)

func main() {
    // Create a new IPStat with a 5-minute window
    stat := ipstat.NewIPStat(5 * time.Minute)
    defer stat.Stop() // Always remember to stop!
    
    // Add IP addresses
    stat.Add("192.168.1.1")
    stat.Add("192.168.1.2")
    stat.Add("192.168.1.1") // Duplicate - won't be counted twice
    
    // Get the count of unique IPs
    count := stat.Count()
    fmt.Printf("Unique IPs: %d\n", count)
}
```

## API

### `NewIPStat(window time.Duration) *IPStat`

Creates a new IPStat instance with the specified time window. The window must be at least 1 minute and will be truncated to minute boundaries.

**Note**: A background goroutine is started automatically. Remember to call `Stop()` to clean it up.

### `Add(ip string)`

Adds an IP address to the statistics. This operation is non-blocking and thread-safe.

### `Count() int64`

Returns the current count of unique IP addresses within the time window.

### `Stop()`

Stops the background goroutine and cleans up resources. Always call this when done.

## How It Works

The library divides the time window into 1-minute buckets. As time progresses:
1. New IPs are added to the current bucket
2. Every minute, the oldest bucket is discarded and the current bucket is moved into the sliding window
3. The count is updated to reflect only IPs seen within the active window

This approach provides accurate unique IP counts while maintaining constant memory usage relative to the window size.
