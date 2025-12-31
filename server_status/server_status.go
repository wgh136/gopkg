package serverstatus

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// collector collects server status metrics
type collector struct {
	// CPU metrics
	cpuUsage *prometheus.Desc

	// Memory metrics
	memoryUsage *prometheus.Desc
	swapUsage   *prometheus.Desc

	// Network metrics
	networkReceiveBytes  *prometheus.Desc
	networkTransmitBytes *prometheus.Desc

	// Disk IOPS metrics
	diskReadOps  *prometheus.Desc
	diskWriteOps *prometheus.Desc

	// Server name label
	serverName string

	// Previous network stats for calculating rates
	prevNetStats map[string]net.IOCountersStat
	prevNetTime  time.Time
	mu           sync.Mutex

	// Previous disk stats for calculating IOPS
	prevDiskStats map[string]disk.IOCountersStat
	prevDiskTime  time.Time
}

// newCollector creates a new server status collector
func newCollector(serverName string) *collector {
	labels := []string{"server_name"}
	return &collector{
		cpuUsage: prometheus.NewDesc(
			"server_cpu_usage_percent",
			"CPU usage percentage",
			labels, nil,
		),
		memoryUsage: prometheus.NewDesc(
			"server_memory_usage_percent",
			"Memory usage percentage",
			labels, nil,
		),
		swapUsage: prometheus.NewDesc(
			"server_swap_usage_percent",
			"Swap usage percentage",
			labels, nil,
		),
		networkReceiveBytes: prometheus.NewDesc(
			"server_network_receive_bytes_per_second",
			"Network receive rate in bytes per second",
			labels, nil,
		),
		networkTransmitBytes: prometheus.NewDesc(
			"server_network_transmit_bytes_per_second",
			"Network transmit rate in bytes per second",
			labels, nil,
		),
		diskReadOps: prometheus.NewDesc(
			"server_disk_read_ops_per_second",
			"Disk read operations per second",
			labels, nil,
		),
		diskWriteOps: prometheus.NewDesc(
			"server_disk_write_ops_per_second",
			"Disk write operations per second",
			labels, nil,
		),
		serverName:    serverName,
		prevNetStats:  make(map[string]net.IOCountersStat),
		prevNetTime:   time.Now(),
		prevDiskStats: make(map[string]disk.IOCountersStat),
		prevDiskTime:  time.Now(),
	}
}

// Describe implements prometheus.Collector
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpuUsage
	ch <- c.memoryUsage
	ch <- c.swapUsage
	ch <- c.networkReceiveBytes
	ch <- c.networkTransmitBytes
	ch <- c.diskReadOps
	ch <- c.diskWriteOps
}

// Collect implements prometheus.Collector
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	// Collect CPU usage
	c.collectCPU(ch)

	// Collect memory and swap usage
	c.collectMemory(ch)

	// Collect network traffic
	c.collectNetwork(ch)

	// Collect disk IOPS
	c.collectDiskIOPS(ch)
}

// collectCPU collects CPU usage metrics
func (c *collector) collectCPU(ch chan<- prometheus.Metric) {
	// Get CPU usage percentage
	percentages, err := cpu.Percent(0, false)
	if err != nil || len(percentages) == 0 {
		// Fallback to reading from /proc/stat
		usage := c.getCPUUsageFromProc()
		ch <- prometheus.MustNewConstMetric(
			c.cpuUsage,
			prometheus.GaugeValue,
			usage,
			c.serverName,
		)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.cpuUsage,
		prometheus.GaugeValue,
		percentages[0],
		c.serverName,
	)
}

// getCPUUsageFromProc reads CPU usage from /proc/stat as fallback
func (c *collector) getCPUUsageFromProc() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0
	}

	line := scanner.Text()
	if !strings.HasPrefix(line, "cpu ") {
		return 0
	}

	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0
	}

	var total, idle uint64
	for i := 1; i < len(fields); i++ {
		val, _ := strconv.ParseUint(fields[i], 10, 64)
		total += val
		if i == 4 { // idle time is the 4th field
			idle = val
		}
	}

	if total == 0 {
		return 0
	}

	return float64(total-idle) / float64(total) * 100
}

// collectMemory collects memory and swap usage metrics
func (c *collector) collectMemory(ch chan<- prometheus.Metric) {
	// Get virtual memory stats
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			c.memoryUsage,
			prometheus.GaugeValue,
			vmStat.UsedPercent,
			c.serverName,
		)
	}

	// Get swap memory stats
	swapStat, err := mem.SwapMemory()
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			c.swapUsage,
			prometheus.GaugeValue,
			swapStat.UsedPercent,
			c.serverName,
		)
	}
}

// isVirtualInterface checks if a network interface is virtual
func (c *collector) isVirtualInterface(name string) bool {
	// Common virtual interface patterns
	virtualPrefixes := []string{
		"lo",      // loopback
		"docker",  // Docker bridge
		"veth",    // Virtual Ethernet (Docker, K8s)
		"br-",     // Linux bridge
		"virbr",   // libvirt bridge
		"vnet",    // Virtual network (KVM/QEMU)
		"tun",     // TUN devices (VPN)
		"tap",     // TAP devices
		"cni",     // Container Network Interface (K8s)
		"flannel", // Flannel network (K8s)
		"calico",  // Calico network (K8s)
		"weave",   // Weave network (K8s)
		"kube",    // Kubernetes interfaces
		"cilium",  // Cilium network (K8s)
		"lxc",     // LXC containers
		"lxd",     // LXD containers
		"vmbr",    // Proxmox bridge
		"vmnet",   // VMware network
		"ppp",     // Point-to-Point Protocol
	}

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// collectNetwork collects network traffic metrics
func (c *collector) collectNetwork(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get network IO counters
	netStats, err := net.IOCounters(true)
	if err != nil {
		return
	}

	now := time.Now()
	timeDiff := now.Sub(c.prevNetTime).Seconds()

	if timeDiff <= 0 {
		return
	}

	var totalReceiveRate, totalTransmitRate float64

	for _, stat := range netStats {
		// Skip loopback and virtual interfaces
		if c.isVirtualInterface(stat.Name) {
			continue
		}

		prevStat, exists := c.prevNetStats[stat.Name]
		if exists {
			receiveRate := float64(stat.BytesRecv-prevStat.BytesRecv) / timeDiff
			transmitRate := float64(stat.BytesSent-prevStat.BytesSent) / timeDiff

			totalReceiveRate += receiveRate
			totalTransmitRate += transmitRate
		}

		c.prevNetStats[stat.Name] = stat
	}

	c.prevNetTime = now

	ch <- prometheus.MustNewConstMetric(
		c.networkReceiveBytes,
		prometheus.GaugeValue,
		totalReceiveRate,
		c.serverName,
	)

	ch <- prometheus.MustNewConstMetric(
		c.networkTransmitBytes,
		prometheus.GaugeValue,
		totalTransmitRate,
		c.serverName,
	)
}

// collectDiskIOPS collects disk IOPS metrics
func (c *collector) collectDiskIOPS(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get disk IO counters
	diskStats, err := disk.IOCounters()
	if err != nil {
		return
	}

	now := time.Now()
	timeDiff := now.Sub(c.prevDiskTime).Seconds()

	if timeDiff <= 0 {
		return
	}

	var totalReadOps, totalWriteOps float64

	for name, stat := range diskStats {
		// Skip loop devices and other virtual devices
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}

		prevStat, exists := c.prevDiskStats[name]
		if exists {
			readOps := float64(stat.ReadCount-prevStat.ReadCount) / timeDiff
			writeOps := float64(stat.WriteCount-prevStat.WriteCount) / timeDiff

			totalReadOps += readOps
			totalWriteOps += writeOps
		}

		c.prevDiskStats[name] = stat
	}

	c.prevDiskTime = now

	ch <- prometheus.MustNewConstMetric(
		c.diskReadOps,
		prometheus.GaugeValue,
		totalReadOps,
		c.serverName,
	)

	ch <- prometheus.MustNewConstMetric(
		c.diskWriteOps,
		prometheus.GaugeValue,
		totalWriteOps,
		c.serverName,
	)
}

// Register registers the server status collector with the provided Prometheus registerer
// If registerer is nil, it will use the default Prometheus registry
// serverName is used as a label for all metrics to identify the server
func Register(registerer prometheus.Registerer, serverName string) error {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	collector := newCollector(serverName)
	return registerer.Register(collector)
}
