package app

import (
	"testing"
	"time"

	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestProcessRuntimeLabels(t *testing.T) {
	adapter, err := newPlatformAdapter()
	require.NoError(t, err)
	report, err := buildReport([]*Process{
		{PID: 1, Name: "pi", Executable: "/usr/local/bin/node"},
		{PID: 2, PPID: 1, Name: "zsh", Executable: "/bin/zsh"},
	}, "1", time.Second, adapter)
	require.NoError(t, err)
	view := display{detail: true, colors: fang.DefaultColorScheme(lipgloss.LightDark(true))}
	output := ansi.Strip(view.render(report))
	require.Contains(t, output, "pi (node)")
	require.NotContains(t, output, "zsh (zsh)")
}
