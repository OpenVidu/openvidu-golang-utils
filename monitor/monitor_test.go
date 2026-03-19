package monitor

import (
	"runtime"
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	m := NewMonitor()
	if m == nil {
		t.Fatal("NewMonitor() returned nil")
	}
}

func TestGetHostCpuIdle_InitialValue(t *testing.T) {
	m := NewMonitor()
	// Before MonitorHostCpu is called the stored value is the zero value.
	if got := m.GetHostCpuIdle(); got != 0 {
		t.Errorf("expected initial GetHostCpuIdle() == 0, got %v", got)
	}
}

func TestMonitorHostCpu(t *testing.T) {
	m := NewMonitor()
	go m.MonitorHostCpu()

	// Give MonitorHostCpu time to initialise (sets hostCpuIdle to numCPU)
	// and then perform at least one ticker tick.
	time.Sleep(2500 * time.Millisecond)

	numCPU := float64(runtime.NumCPU())
	idle := m.GetHostCpuIdle()

	if idle < 0 || idle > numCPU {
		t.Errorf("GetHostCpuIdle() = %v; want value in [0, %v]", idle, numCPU)
	}
}
