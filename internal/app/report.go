package app

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Process contains this process's measurements, never its descendants' totals.
type Process struct {
	PID            int32      `json:"pid"`
	PPID           int32      `json:"ppid"`
	Name           string     `json:"name"`
	Executable     string     `json:"executable,omitempty"`
	Command        string     `json:"command,omitempty"`
	CPUPercent     *float64   `json:"cpu_percent"`
	FootprintBytes *uint64    `json:"footprint_bytes"`
	Children       []*Process `json:"children,omitempty"`
}

type Group struct {
	Name             string     `json:"name"`
	CPUPercent       *float64   `json:"cpu_percent"`
	FootprintBytes   *uint64    `json:"footprint_bytes"`
	CPUIncomplete    bool       `json:"cpu_incomplete"`
	MemoryIncomplete bool       `json:"memory_incomplete"`
	ProcessCount     int        `json:"process_count"`
	Processes        []*Process `json:"processes"`
}

type Report struct {
	SampledAt     time.Time `json:"sampled_at"`
	SampleSeconds float64   `json:"sample_interval_seconds"`
	MemoryMetric  string    `json:"memory_metric"`
	Groups        []Group   `json:"groups"`
}

func appBundle(path string) string {
	if i := strings.Index(path, ".app/"); i >= 0 {
		return path[:i+4]
	}
	return ""
}

// buildReport assigns every PID to one group before calculating totals.
// A descendant belongs to the outermost app in its ancestry, even if it is
// another app or its executable lives outside that app's bundle.
func buildReport(processes []*Process, target string, interval time.Duration) (Report, error) {
	byPID := make(map[int32]*Process, len(processes))
	for _, p := range processes {
		p.Children = nil
		byPID[p.PID] = p
	}

	selectedPID, pidErr := strconv.ParseInt(target, 10, 32)
	if pidErr == nil && selectedPID <= 0 {
		return Report{}, fmt.Errorf("PID must be positive")
	}
	if pidErr == nil && byPID[int32(selectedPID)] == nil {
		return Report{}, fmt.Errorf("process %d is not running or cannot be read", selectedPID)
	}

	members := make(map[string][]*Process)
	names := make(map[string]string)
	for _, p := range byPID {
		owner := ""
		seen := make(map[int32]bool)
		for ancestor := p; ancestor != nil && !seen[ancestor.PID]; ancestor = byPID[ancestor.PPID] {
			seen[ancestor.PID] = true
			if pidErr == nil && ancestor.PID == int32(selectedPID) {
				owner = "pid:" + target
				break
			}
			if bundle := appBundle(ancestor.Executable); bundle != "" {
				owner = bundle
			}
		}
		if pidErr == nil && owner != "pid:"+target {
			continue
		}
		if owner == "" {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(owner), ".app")
		if pidErr == nil {
			name = byPID[int32(selectedPID)].Name
		}
		if pidErr != nil && !strings.Contains(strings.ToLower(name), strings.ToLower(target)) {
			continue
		}
		members[owner] = append(members[owner], p)
		names[owner] = name
	}

	report := Report{
		SampledAt: time.Now().UTC(), SampleSeconds: interval.Seconds(),
		MemoryMetric: "physical_footprint", Groups: []Group{},
	}
	for owner, groupProcesses := range members {
		g := Group{Name: names[owner], ProcessCount: len(groupProcesses), Processes: []*Process{}}
		inGroup := make(map[int32]bool, len(groupProcesses))
		for _, p := range groupProcesses {
			inGroup[p.PID] = true
			if p.CPUPercent == nil {
				g.CPUIncomplete = true
			} else {
				if g.CPUPercent == nil {
					g.CPUPercent = new(float64)
				}
				*g.CPUPercent += *p.CPUPercent
			}
			if p.FootprintBytes == nil {
				g.MemoryIncomplete = true
			} else {
				if g.FootprintBytes == nil {
					g.FootprintBytes = new(uint64)
				}
				*g.FootprintBytes += *p.FootprintBytes
			}
		}
		// A cycle cannot occur in a live process tree. Drop malformed parent
		// links rather than creating a recursive JSON/rendering structure.
		for _, p := range groupProcesses {
			parent := byPID[p.PPID]
			seen := map[int32]bool{p.PID: true}
			cycle := false
			for a := parent; a != nil && inGroup[a.PID]; a = byPID[a.PPID] {
				if seen[a.PID] {
					cycle = true
					break
				}
				seen[a.PID] = true
			}
			if parent != nil && inGroup[parent.PID] && !cycle {
				parent.Children = append(parent.Children, p)
			} else {
				g.Processes = append(g.Processes, p)
			}
		}
		for _, p := range groupProcesses {
			sortProcesses(p.Children)
		}
		sortProcesses(g.Processes)
		report.Groups = append(report.Groups, g)
	}
	slices.SortFunc(report.Groups, func(a, b Group) int {
		var left, right uint64
		if a.FootprintBytes != nil {
			left = *a.FootprintBytes
		}
		if b.FootprintBytes != nil {
			right = *b.FootprintBytes
		}
		if order := cmp.Compare(right, left); order != 0 {
			return order
		}
		return strings.Compare(a.Name, b.Name)
	})
	if target != "" && len(report.Groups) == 0 {
		return Report{}, fmt.Errorf("no running app matches %q", target)
	}
	return report, nil
}

func sortProcesses(processes []*Process) {
	slices.SortFunc(processes, func(a, b *Process) int { return cmp.Compare(a.PID, b.PID) })
}
