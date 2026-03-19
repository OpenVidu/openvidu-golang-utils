package monitor

import (
	"math"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	l, _ := zap.NewProduction()
	logger = l.Sugar()
}

// atomicFloat64 is an atomic float64 value using uint64 bit representation.
type atomicFloat64 struct {
	v atomic.Uint64
}

func (f *atomicFloat64) Store(val float64) {
	f.v.Store(math.Float64bits(val))
}

func (f *atomicFloat64) Load() float64 {
	return math.Float64frombits(f.v.Load())
}

// Monitor tracks host resource metrics.
type Monitor struct {
	hostCpuIdle atomicFloat64
}

// NewMonitor creates and returns a new Monitor instance.
func NewMonitor() *Monitor {
	return &Monitor{}
}

// GetHostCpuIdle returns the latest measured host CPU idle value.
// The value represents the number of idle CPUs (0 to runtime.NumCPU()).
func (m *Monitor) GetHostCpuIdle() float64 {
	return m.hostCpuIdle.Load()
}

// MonitorHostCpu continuously measures host CPU idle and stores it
// in the Monitor. It blocks indefinitely and should be run in a goroutine.
func (m *Monitor) MonitorHostCpu() {
	numCPU := float64(runtime.NumCPU())
	prev, _ := cpu.Get()
	m.hostCpuIdle.Store(numCPU)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		next, err := cpu.Get()
		if err != nil {
			logger.Errorw("failed retrieving host CPU idle", "error", err)
			continue
		}
		if d := next.Total - prev.Total; d > 0 {
			idle := numCPU * float64(next.Idle-prev.Idle) / float64(d)
			m.hostCpuIdle.Store(idle)
		}
		prev = next
	}
}
