package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

// Run owns command configuration; presentation and collection remain separate.
func Run(ctx context.Context) error {
	var watch, jsonOutput, verbose bool
	var depth int
	var interval time.Duration
	cmd := &cobra.Command{
		Use:     "appsum [app-name|pid]",
		Short:   "See an app's CPU usage and physical memory footprint",
		Long:    "CPU and physical memory footprint for macOS apps and their descendants.\n\nWithout a target, lists app totals, largest footprint first.\nWith an app name or PID, shows its total and full process tree.\nCPU: 100% = one busy core. Memory: physical footprint, never RSS.",
		Example: "  appsum\n  appsum asklog\n  appsum 7836\n  appsum asklog --watch\n  appsum asklog --json",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if interval < 100*time.Millisecond {
				return fmt.Errorf("--interval must be at least 100ms")
			}
			if depth < 0 {
				return fmt.Errorf("--depth must be zero or positive")
			}
			target := ""
			if len(args) != 0 {
				target = args[0]
			}
			sample := func(ctx context.Context) (Report, error) {
				processes, err := collectProcesses(ctx, interval, verbose)
				if err != nil {
					return Report{}, err
				}
				return buildReport(processes, target, interval)
			}
			interactive := term.IsTerminal(os.Stdout.Fd()) && !jsonOutput
			view := display{
				colors: fang.DefaultColorScheme(lipgloss.LightDark(true)),
				detail: target != "", verbose: verbose, depth: depth,
			}
			if interactive {
				view.width, _, _ = term.GetSize(os.Stdout.Fd())
				view.colors = fang.DefaultColorScheme(lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout)))
			}
			if interactive {
				err := runTerminal(cmd.Context(), sample, view, watch)
				if cmd.Context().Err() != nil {
					return nil
				}
				return err
			}
			output := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
			for {
				report, err := sample(cmd.Context())
				if err != nil {
					if cmd.Context().Err() != nil {
						return nil
					}
					return err
				}
				if jsonOutput {
					err = json.NewEncoder(cmd.OutOrStdout()).Encode(report)
				} else {
					_, err = fmt.Fprintln(output, view.render(report))
				}
				if err != nil {
					return err
				}
				if !watch {
					return nil
				}
			}
		},
	}
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Refresh in place; stream snapshots when piped")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print JSON; --watch emits newline-delimited JSON")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show full names, executable paths, and command lines")
	cmd.Flags().DurationVar(&interval, "interval", time.Second, "CPU sampling interval")
	cmd.Flags().IntVar(&depth, "depth", 0, "Limit displayed process depth; 0 shows every descendant (JSON is always complete)")
	return fang.Execute(ctx, cmd, fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM))
}
