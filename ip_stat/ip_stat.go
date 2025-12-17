package ipstat

import (
	"sync/atomic"
	"time"

	"github.com/wgh136/gopkg/ip_stat/internal"
)

const (
	bucketSize = 1 * time.Minute
)

// IPStat is used to count the number of unique IPs in a given window. It is thread-safe.
type IPStat struct {
	count   map[string]int64
	window  time.Duration
	buckets []*bucket
	current *bucket
	index   int
	ipChan  chan string
	stat    atomic.Int64
}

type bucket struct {
	ips *internal.Set[string]
}

func (b *bucket) add(ip string) {
	b.ips.Add(ip)
}

// NewIPStat creates a new IPStat with the given window size. A new goroutine is started to maintain the IPStat. Remember to call Stop() to clean up the goroutine.
func NewIPStat(window time.Duration) *IPStat {
	if window < bucketSize {
		panic("window size is too small")
	}
	window = window.Truncate(bucketSize)
	numBuckets := int(window / bucketSize)
	buckets := make([]*bucket, numBuckets)
	for i := 0; i < numBuckets; i++ {
		buckets[i] = &bucket{
			ips: internal.NewSet[string](),
		}
	}
	stat := &IPStat{
		count:   make(map[string]int64),
		window:  window,
		buckets: buckets,
		current: &bucket{
			ips: internal.NewSet[string](),
		},
		ipChan: make(chan string, 100),
	}
	stat.runMaintainer()
	return stat
}

// Add adds an IP to the IPStat.
func (s *IPStat) Add(ip string) {
	s.ipChan <- ip
}

func (s *IPStat) runMaintainer() {
	go func() {
		ticker := time.NewTicker(bucketSize)
		for {
			select {
			case <-ticker.C:
				s.buckets[s.index].ips.Range(func(ip string) bool {
					s.count[ip]--
					if s.count[ip] == 0 {
						delete(s.count, ip)
					}
					return true
				})
				s.current.ips.Range(func(ip string) bool {
					s.count[ip]++
					return true
				})
				s.buckets[s.index] = s.current
				s.index = (s.index + 1) % len(s.buckets)
				s.current = &bucket{
					ips: internal.NewSet[string](),
				}
				s.stat.Store(int64(len(s.count)))
			case ip, ok := <-s.ipChan:
				if !ok {
					ticker.Stop()
					return
				}
				s.current.add(ip)
			}
		}
	}()
}

// Stop stops the IPStat.
func (s *IPStat) Stop() {
	close(s.ipChan)
}

// Count returns the number of unique IPs in the IPStat.
func (s *IPStat) Count() int64 {
	return s.stat.Load()
}
