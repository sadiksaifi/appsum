package app

import (
	"context"
	"strconv"

	"github.com/shirou/gopsutil/v4/process"
)

const (
	memoryMetric = "pss"
	memoryLabel  = "PSS (proportional set size)"
)

type linuxAdapter struct{}

func newPlatformAdapter() (platformAdapter, error) { return linuxAdapter{}, nil }

// Linux CPU reads already return errors through gopsutil.
func (linuxAdapter) cpuReadable(_ *process.Process) bool { return true }

func (linuxAdapter) memory(ctx context.Context, p *process.Process) *uint64 {
	// Grouped maps use smaps_rollup where available; Pss is in kernel KiB.
	maps, err := p.MemoryMapsWithContext(ctx, true)
	if err != nil || maps == nil || len(*maps) == 0 {
		return nil
	}
	memory := (*maps)[0]
	if memory.Pss == 0 && memory.Rss == 0 {
		return nil // An empty address space is not evidence of a readable PSS.
	}
	bytes := memory.Pss * 1024
	return &bytes
}

// A standalone tree starts immediately below init or at a missing parent.
func (linuxAdapter) group(p *Process, byPID map[int32]*Process) groupKey {
	root := p
	seen := make(map[int32]bool)
	for a := p; a != nil && a.PID > 1 && !seen[a.PID]; a = byPID[a.PPID] {
		seen[a.PID] = true
		root = a
	}
	return groupKey{strconv.Itoa(int(root.PID)), root.Name}
}
