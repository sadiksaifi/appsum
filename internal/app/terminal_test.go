package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/require"
)

func TestVerboseWatchRejectsOverwideFrames(t *testing.T) {
	report := Report{Groups: []Group{{Name: "worker", ProcessCount: 1, Processes: []*Process{
		{PID: 100, Name: "worker", Command: "worker " + strings.Repeat("x", 180)},
	}}}}
	for _, resized := range []bool{false, true} {
		name := "initial sample"
		if resized {
			name = "terminal resize"
		}
		t.Run(name, func(t *testing.T) {
			model := terminal{watch: true, height: 24, display: display{
				detail: true, verbose: true, width: 80,
				colors: fang.DefaultColorScheme(lipgloss.LightDark(true)),
			}}
			var message tea.Msg = sampleResult{report: report}
			if resized {
				model.report = &report
				model.display.width = 300
				message = tea.WindowSizeMsg{Width: 80, Height: 24}
			}
			updated, command := model.Update(message)
			require.Empty(t, updated.View().Content)
			require.NotNil(t, command)
			require.IsType(t, tea.QuitMsg{}, command())
		})
	}
}
