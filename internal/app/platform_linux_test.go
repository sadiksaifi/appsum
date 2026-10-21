package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStandaloneTreesStaySeparate(t *testing.T) {
	report, err := buildReport([]*Process{
		{PID: 10, PPID: 1, Name: "service", MemoryBytes: new(uint64(30))},
		{PID: 11, PPID: 10, Name: "worker", MemoryBytes: new(uint64(20))},
		{PID: 12, PPID: 11, Name: "child", MemoryBytes: new(uint64(10))},
		{PID: 20, PPID: 1, Name: "other", MemoryBytes: new(uint64(5))},
	}, "", time.Second, linuxAdapter{})
	require.NoError(t, err)
	require.Len(t, report.Groups, 2)
	require.Equal(t, "service", report.Groups[0].Name)
	require.Equal(t, 3, report.Groups[0].ProcessCount)
	require.Equal(t, new(uint64(60)), report.Groups[0].MemoryBytes)
	require.Equal(t, int32(12), report.Groups[0].Processes[0].Children[0].Children[0].PID)
	require.Equal(t, "other", report.Groups[1].Name)
}
