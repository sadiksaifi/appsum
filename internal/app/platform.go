package app

import (
	"context"

	"github.com/shirou/gopsutil/v4/process"
)

// platformAdapter owns OS-specific accounting and process ownership. Sampling,
// totals, rendering, and command handling do not depend on native APIs.
type platformAdapter interface {
	memory(context.Context, *process.Process) *uint64
	cpuReadable(*process.Process) bool
	group(*Process, map[int32]*Process) groupKey
}
