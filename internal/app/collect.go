package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

func collectProcesses(ctx context.Context, interval time.Duration, verbose bool, adapter platformAdapter) ([]*Process, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if interval <= 0 {
		return nil, fmt.Errorf("sampling interval must be positive")
	}
	first, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("enumerate processes: %w", err)
	}
	baseline := make(map[int32]*process.Process, len(first))
	for _, p := range first {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		created, err := p.CreateTimeWithContext(ctx)
		if err != nil || created <= 0 || !adapter.cpuReadable(p) {
			continue
		}
		if _, err := p.PercentWithContext(ctx, 0); err == nil {
			baseline[p.Pid] = p
		}
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}

	second, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("enumerate processes: %w", err)
	}
	result := make([]*Process, 0, len(second))
	for _, p := range second {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		snapshot := &Process{PID: p.Pid}
		// Compare against a fresh object: CreateTime is cached by gopsutil.
		if previous := baseline[p.Pid]; previous != nil {
			created, err := p.CreateTimeWithContext(ctx)
			before, beforeErr := previous.CreateTimeWithContext(ctx)
			if err == nil && beforeErr == nil && created == before && adapter.cpuReadable(p) {
				if percent, err := previous.PercentWithContext(ctx, 0); err == nil {
					snapshot.CPUPercent = &percent
				}
			}
		}
		snapshot.MemoryBytes = adapter.memory(ctx, p)
		snapshot.PPID, _ = p.PpidWithContext(ctx)
		snapshot.Executable, _ = p.ExeWithContext(ctx)
		args, _ := p.CmdlineSliceWithContext(ctx)
		// argv[0] preserves titles such as "pi"; Exe still identifies Node.
		if len(args) > 0 && args[0] != "" {
			snapshot.Name = filepath.Base(args[0])
		} else {
			snapshot.Name, _ = p.NameWithContext(ctx)
		}
		if snapshot.Name == "" && snapshot.Executable != "" {
			snapshot.Name = filepath.Base(snapshot.Executable)
		}
		if snapshot.Name == "" {
			snapshot.Name = strconv.Itoa(int(p.Pid))
		}
		if verbose {
			snapshot.Command = strings.Join(args, " ")
		}
		result = append(result, snapshot)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
