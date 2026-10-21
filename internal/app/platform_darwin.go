package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/shirou/gopsutil/v4/process"
)

const (
	memoryMetric = "physical_footprint"
	memoryLabel  = "physical footprint"
)

// Apple's stable rusage_info_v0 ABI from sys/resource.h: 96 bytes on both
// supported architectures. PhysicalFootprint is the ledger value, not RSS.
type rusageInfoV0 struct {
	UUID                                                [16]byte
	UserTime, SystemTime, IdleWakeups, InterruptWakeups uint64
	Pageins, WiredSize, ResidentSize, PhysicalFootprint uint64
	StartTime, ExitTime                                 uint64
}

type darwinAdapter struct {
	rusage   func(int32, int32, unsafe.Pointer) int32
	taskInfo func(int32, int32, uint64, unsafe.Pointer, int32) int32
}

func newPlatformAdapter() (platformAdapter, error) {
	// Keep the library loaded for the lifetime of its registered functions.
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_LOCAL)
	if err != nil {
		return nil, fmt.Errorf("load macOS process accounting: %w", err)
	}
	var adapter darwinAdapter
	purego.RegisterLibFunc(&adapter.rusage, lib, "proc_pid_rusage")
	purego.RegisterLibFunc(&adapter.taskInfo, lib, "proc_pidinfo")
	return adapter, nil
}

func (a darwinAdapter) memory(_ context.Context, p *process.Process) *uint64 {
	var usage rusageInfoV0
	if a.rusage(p.Pid, 0, unsafe.Pointer(&usage)) != 0 {
		return nil
	}
	return &usage.PhysicalFootprint
}

// gopsutil's Darwin Times method does not check proc_pidinfo's return value.
func (a darwinAdapter) cpuReadable(p *process.Process) bool {
	var info process.ProcTaskInfo
	size := int32(unsafe.Sizeof(info))
	return a.taskInfo(p.Pid, 4, 0, unsafe.Pointer(&info), size) == size
}

func appBundle(path string) string {
	if i := strings.Index(path, ".app/"); i >= 0 {
		return path[:i+4]
	}
	return ""
}

func (darwinAdapter) group(p *Process, byPID map[int32]*Process) groupKey {
	var bundle string
	seen := make(map[int32]bool)
	for a := p; a != nil && !seen[a.PID]; a = byPID[a.PPID] {
		seen[a.PID] = true
		if path := appBundle(a.Executable); path != "" {
			bundle = path
		}
	}
	if bundle == "" {
		return groupKey{}
	}
	return groupKey{bundle, strings.TrimSuffix(filepath.Base(bundle), ".app")}
}
