package monitor

import (
	"context"
	"math"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
)

// Logger defines the logging interface used by the monitor package.
// It is compatible with livekit/protocol/logger and other structured loggers.
type Logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, err error, keysAndValues ...interface{})
	Errorw(msg string, err error, keysAndValues ...interface{})
}

// noopLogger discards all log output.
type noopLogger struct{}

func (noopLogger) Infow(string, ...interface{})         {}
func (noopLogger) Warnw(string, error, ...interface{})  {}
func (noopLogger) Errorw(string, error, ...interface{}) {}

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
	cancel      context.CancelFunc
	logger      Logger
}

// Option configures a Monitor.
type Option func(*Monitor)

// WithLogger sets a custom logger. If not provided, logging is disabled.
func WithLogger(l Logger) Option {
	return func(m *Monitor) {
		if l != nil {
			m.logger = l
		}
	}
}

// NewMonitor creates and returns a new Monitor instance.
func NewMonitor(opts ...Option) *Monitor {
	m := &Monitor{
		logger: noopLogger{},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// GetHostCpuIdle returns the latest measured host CPU idle value.
// The value represents the number of idle CPUs (0 to runtime.NumCPU()).
func (m *Monitor) GetHostCpuIdle() float64 {
	return m.hostCpuIdle.Load()
}

// Start begins host CPU monitoring in a background goroutine.
// It is safe to call from any goroutine. Call Stop to release resources.
func (m *Monitor) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.monitorHostCpu(ctx)
}

// Stop terminates the background monitoring goroutine started by Start.
// It is safe to call even if Start was never called.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// monitorHostCpu continuously measures host CPU idle until the context is cancelled.
func (m *Monitor) monitorHostCpu(ctx context.Context) {
	numCPU := float64(runtime.NumCPU())

	prev, err := cpu.Get()
	if err != nil {
		m.logger.Errorw("failed retrieving initial host CPU stats", err)
	}
	m.hostCpuIdle.Store(numCPU)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Infow("host CPU monitor stopped")
			return
		case <-ticker.C:
			next, err := cpu.Get()
			if err != nil {
				m.logger.Errorw("failed retrieving host CPU stats", err)
				continue
			}
			if totalDelta := next.Total - prev.Total; totalDelta > 0 {
				idle := numCPU * float64(next.Idle-prev.Idle) / float64(totalDelta)
				// Clamp to [0, numCPU] to guard against kernel counter quirks
				m.hostCpuIdle.Store(math.Max(0, math.Min(numCPU, idle)))
			}
			prev = next
		}
	}
}
