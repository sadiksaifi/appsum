package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProcessTreeTotals(t *testing.T) {
	main := &Process{PID: 100, PPID: 1, Name: "asklog", Executable: "/Applications/asklog.app/Contents/MacOS/asklog", CPUPercent: new(1.0), FootprintBytes: new(uint64(100))}
	child := &Process{PID: 101, PPID: 100, Name: "agent", CPUPercent: new(2.0), FootprintBytes: new(uint64(50))}
	grandchild := &Process{PID: 102, PPID: 101, Name: "other", Executable: "/Applications/Other.app/Contents/MacOS/other", CPUPercent: new(0.5), FootprintBytes: new(uint64(25))}
	unrelated := &Process{PID: 200, PPID: 1, Name: "unrelated", Executable: "/Applications/Unrelated.app/Contents/MacOS/unrelated", CPUPercent: new(5.0), FootprintBytes: new(uint64(400))}
	processes := []*Process{main, child, grandchild, unrelated, child}

	report, err := buildReport(processes, "ASKLOG", time.Second)
	require.NoError(t, err)
	require.Len(t, report.Groups, 1)
	group := report.Groups[0]
	require.Equal(t, 3, group.ProcessCount)
	require.Equal(t, new(3.5), group.CPUPercent)
	require.Equal(t, new(uint64(175)), group.FootprintBytes)
	require.Equal(t, int32(102), group.Processes[0].Children[0].Children[0].PID)

	report, err = buildReport(processes, "101", time.Second)
	require.NoError(t, err)
	require.Len(t, report.Groups, 1)
	require.Equal(t, 2, report.Groups[0].ProcessCount)
	require.Equal(t, new(2.5), report.Groups[0].CPUPercent)
	require.Equal(t, new(uint64(75)), report.Groups[0].FootprintBytes)
}

func TestUnavailableMeasurementsArePartial(t *testing.T) {
	report, err := buildReport([]*Process{
		{PID: 100, PPID: 1, Name: "app", Executable: "/Applications/App.app/Contents/MacOS/app", CPUPercent: new(1.0), FootprintBytes: new(uint64(100))},
		{PID: 101, PPID: 100, Name: "protected"},
	}, "app", time.Second)
	require.NoError(t, err)
	require.Len(t, report.Groups, 1)
	group := report.Groups[0]
	require.True(t, group.CPUIncomplete)
	require.True(t, group.MemoryIncomplete)
	require.Equal(t, new(1.0), group.CPUPercent)
	require.Equal(t, new(uint64(100)), group.FootprintBytes)
	require.Nil(t, group.Processes[0].Children[0].FootprintBytes)

	report, err = buildReport([]*Process{{PID: 200, Name: "unreadable"}}, "200", time.Second)
	require.NoError(t, err)
	require.Nil(t, report.Groups[0].CPUPercent)
	require.Nil(t, report.Groups[0].FootprintBytes)
}
