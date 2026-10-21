//go:build darwin && cgo

package app

/*
#include <libproc.h>
#include <sys/proc_info.h>
#include <sys/resource.h>

static int physical_footprint(int pid, uint64_t *bytes) {
	struct rusage_info_v0 usage;
	int result = proc_pid_rusage(pid, RUSAGE_INFO_V0, (rusage_info_t *)&usage);
	if (result == 0) *bytes = usage.ri_phys_footprint;
	return result;
}

// gopsutil Times does not check whether this native read succeeds.
static int task_readable(int pid) {
	struct proc_taskinfo info;
	return proc_pidinfo(pid, PROC_PIDTASKINFO, 0, &info, sizeof(info)) == sizeof(info);
}
*/
import "C"

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

func collectProcesses(ctx context.Context, interval time.Duration, verbose bool) ([]*Process, error) {
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
		if err != nil || created <= 0 || C.task_readable(C.int(p.Pid)) == 0 {
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
			if err == nil && beforeErr == nil && created == before && C.task_readable(C.int(p.Pid)) != 0 {
				if percent, err := previous.PercentWithContext(ctx, 0); err == nil {
					snapshot.CPUPercent = &percent
				}
			}
		}
		var footprint C.uint64_t
		if C.physical_footprint(C.int(p.Pid), &footprint) == 0 {
			bytes := uint64(footprint)
			snapshot.FootprintBytes = &bytes
		}
		snapshot.PPID, _ = p.PpidWithContext(ctx)
		snapshot.Executable, _ = p.ExeWithContext(ctx)
		if snapshot.Executable != "" {
			snapshot.Name = filepath.Base(snapshot.Executable)
		} else {
			snapshot.Name, _ = p.NameWithContext(ctx)
		}
		if snapshot.Name == "" {
			snapshot.Name = strconv.Itoa(int(p.Pid))
		}
		if verbose {
			snapshot.Command, _ = p.CmdlineWithContext(ctx)
		}
		result = append(result, snapshot)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
