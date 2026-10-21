package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"charm.land/lipgloss/v2/tree"
	"github.com/charmbracelet/x/ansi"
	"github.com/dustin/go-humanize"
)

type display struct {
	colors  fang.ColorScheme
	width   int
	detail  bool
	verbose bool
	depth   int
}

func (d display) render(report Report) string {
	rows := [][]string{}
	totals := make(map[int]bool)
	partial := false
	processCount := 0
	for _, group := range report.Groups {
		totals[len(rows)] = true
		partial = partial || group.CPUIncomplete || group.MemoryIncomplete
		processCount += group.ProcessCount
		cpu := formatCPU(group.CPUPercent, group.CPUIncomplete)
		memory := formatMemory(group.FootprintBytes, group.MemoryIncomplete)
		groupRows := [][]string{{clean(group.Name), "—", cpu, memory}}
		if d.detail {
			root := tree.Root(clean(group.Name)).EnumeratorStyle(lipgloss.NewStyle().Foreground(d.colors.Comment).MarginRight(1))
			var branch func(*Process, int) *tree.Tree
			branch = func(p *Process, level int) *tree.Tree {
				name := p.Name
				if p.Executable != "" {
					if runtime := filepath.Base(p.Executable); !strings.EqualFold(name, runtime) {
						name += " (" + runtime + ")"
					}
				}
				if d.verbose {
					name = p.Command
					if name == "" {
						name = p.Executable
					}
				}
				cpu := formatCPU(p.CPUPercent, false)
				memory := formatMemory(p.FootprintBytes, false)
				groupRows = append(groupRows, []string{"", strconv.Itoa(int(p.PID)), cpu, memory})
				node := tree.Root(clean(name))
				if d.depth == 0 || level < d.depth {
					for _, child := range p.Children {
						node.Child(branch(child, level+1))
					}
				}
				return node
			}
			for _, p := range group.Processes {
				root.Child(branch(p, 1))
			}
			for i, line := range strings.Split(root.String(), "\n") {
				groupRows[i][0] = line
			}
		}
		rows = append(rows, groupRows...)
	}

	headers := []string{"APP / PROCESS", "PID", "CPU", "MEMORY"}
	// Only names may shrink: numeric columns retain their full widths.
	if d.width > 0 && !d.verbose {
		fixed := 8 // Two padding cells per column.
		for col := 1; col < len(headers); col++ {
			width := ansi.StringWidth(headers[col])
			for _, row := range rows {
				width = max(width, ansi.StringWidth(row[col]))
			}
			fixed += width
		}
		nameWidth := max(1, d.width-fixed)
		headers[0] = ansi.Truncate(headers[0], nameWidth, "…")
		for _, row := range rows {
			row[0] = ansi.Truncate(row[0], nameWidth, "…")
		}
	}
	t := table.New().Headers(headers...).Rows(rows...).Wrap(false).
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderHeader(false).BorderColumn(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if col > 0 {
				style = style.Align(lipgloss.Right)
			}
			if row == table.HeaderRow {
				return style.Foreground(d.colors.Comment)
			}
			if totals[row] {
				return style.Bold(true).Foreground(d.colors.Title)
			}
			if col == 1 {
				return style.Foreground(d.colors.Comment)
			}
			if col == 2 {
				return style.Foreground(d.colors.Flag)
			}
			if col == 3 {
				return style.Foreground(d.colors.Program)
			}
			return style
		})
	footer := fmt.Sprintf("Apps: %d · Processes: %d · physical footprint · %s", len(report.Groups), processCount, report.SampledAt.Local().Format("15:04:05"))
	if partial {
		footer += "\n≥ partial total · — unavailable · no RSS substitution"
	}
	if d.depth > 0 && d.detail {
		footer += fmt.Sprintf("\nShowing %d process levels; totals include every descendant.", d.depth)
	}
	if slices.ContainsFunc(report.Groups, func(g Group) bool { return g.CPUIncomplete }) {
		footer += "\nCPU may be unavailable for new or inaccessible processes."
	}
	return t.String() + "\n\n" + lipgloss.NewStyle().Foreground(d.colors.Comment).Render(footer)
}

func formatCPU(value *float64, partial bool) string {
	if value == nil {
		return "—"
	}
	prefix := ""
	if partial {
		prefix = "≥"
	}
	return fmt.Sprintf("%s%.1f%%", prefix, *value)
}

func formatMemory(value *uint64, partial bool) string {
	if value == nil {
		return "—"
	}
	prefix := ""
	if partial {
		prefix = "≥"
	}
	return prefix + humanize.IBytes(*value)
}

func clean(s string) string {
	return strings.Join(strings.Fields(ansi.Strip(s)), " ")
}
