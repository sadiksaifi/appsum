package app

import (
	"context"
	"fmt"
	"os"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

type sampleResult struct {
	report Report
	err    error
}

type terminal struct {
	ctx      context.Context
	sample   func(context.Context) (Report, error)
	display  display
	spinner  spinner.Model
	report   *Report
	watch    bool
	height   int
	overflow bool
	err      error
}

func (m terminal) collect() tea.Msg {
	report, err := m.sample(m.ctx)
	return sampleResult{report, err}
}

func (m terminal) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.collect)
}

func (m terminal) overflows() bool {
	if m.report == nil {
		return false
	}
	rendered := m.display.render(*m.report)
	return (m.display.width > 0 && lipgloss.Width(rendered) > m.display.width) ||
		(m.height > 0 && lipgloss.Height(rendered)+1 >= m.height)
}

func (m terminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sampleResult:
		m.err = msg.err
		if msg.err != nil {
			return m, tea.Quit
		}
		m.report = &msg.report
		m.overflow = m.overflows()
		if !m.watch || m.overflow {
			return m, tea.Quit
		}
		return m, m.collect
	case tea.WindowSizeMsg:
		m.display.width, m.height = msg.Width, msg.Height
		m.overflow = m.overflows()
		if m.watch && m.overflow {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		if m.report == nil {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m terminal) View() tea.View {
	if m.report == nil && m.err == nil {
		return tea.NewView(m.spinner.View() + " Measuring CPU and " + memoryLabel + "…")
	}
	if m.report == nil || !m.watch || m.overflow {
		return tea.NewView("")
	}
	return tea.NewView(m.display.render(*m.report) + "\nCtrl+C to stop")
}

func runTerminal(ctx context.Context, sample func(context.Context) (Report, error), view display, watch bool) error {
	model := terminal{ctx: ctx, sample: sample, display: view, watch: watch, spinner: spinner.New()}
	result, err := tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(os.Stdout)).Run()
	if err != nil && ctx.Err() == nil {
		return err
	}
	final, ok := result.(terminal)
	if !ok {
		return err
	}
	if final.err != nil && ctx.Err() == nil {
		return final.err
	}
	output := colorprofile.NewWriter(os.Stdout, os.Environ())
	if final.report != nil && (!watch || final.overflow) {
		if _, err := fmt.Fprintln(output, final.display.render(*final.report)); err != nil {
			return err
		}
	}
	if final.overflow && watch && ctx.Err() == nil {
		if _, err := fmt.Fprintln(os.Stderr, "Report exceeds terminal size; streaming full snapshots. Widen the terminal, use --depth, or omit --verbose for in-place updates."); err != nil {
			return err
		}
		for ctx.Err() == nil {
			report, err := sample(ctx)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(output, final.display.render(report)); err != nil {
				return err
			}
		}
	}
	return ctx.Err()
}
