package metrics

import (
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

type SystemCollector struct {
	process *process.Process
}

func NewSystemCollector() *SystemCollector {
	proc, _ := process.NewProcess(int32(os.Getpid()))
	return &SystemCollector{process: proc}
}

func (c *SystemCollector) Snapshot(queueDepth, bufferedEvents int) SystemStats {
	stats := SystemStats{
		Goroutines:     runtime.NumGoroutine(),
		QueueDepth:     queueDepth,
		BufferedEvents: bufferedEvents,
	}

	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		stats.CPUPercent = pct[0]
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		stats.MemoryTotalMB = vm.Total / 1024 / 1024
		stats.MemoryUsagePct = vm.UsedPercent
	}

	if c.process != nil {
		if mi, err := c.process.MemoryInfo(); err == nil {
			stats.MemoryUsedMB = mi.RSS / 1024 / 1024
		}
		if fds, err := c.process.NumFDs(); err == nil {
			stats.OpenFileDesc = uint64(fds)
		}
	}

	return stats
}
