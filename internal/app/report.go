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
	PID         int32      `json:"pid"`
	PPID        int32      `json:"ppid"`
	Name        string     `json:"name"`
	Executable  string     `json:"executable,omitempty"`
	Command     string     `json:"command,omitempty"`
	CPUPercent  *float64   `json:"cpu_percent"`
	MemoryBytes *uint64    `json:"memory_bytes"`
	Children    []*Process `json:"children,omitempty"`
}

type Group struct {
	Name             string     `json:"name"`
	CPUPercent       *float64   `json:"cpu_percent"`
	MemoryBytes      *uint64    `json:"memory_bytes"`
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

type groupKey struct{ id, name string }

func selectGroups(byPID map[int32]*Process, target string, adapter platformAdapter) (map[groupKey][]*Process, error) {
	members := make(map[groupKey][]*Process)
	roots := make(map[int32]bool)
	pid, pidErr := strconv.ParseInt(target, 10, 32)
	if pidErr == nil {
		if pid <= 0 {
			return nil, fmt.Errorf("PID must be positive")
		}
		if byPID[int32(pid)] == nil {
			return nil, fmt.Errorf("process %d is not running or cannot be read", pid)
		}
		roots[int32(pid)] = true
	} else {
		query := strings.ToLower(target)
		for _, exact := range []bool{true, false} {
			matches := func(name string) bool {
				name = strings.ToLower(name)
				return target == "" || name == query || (!exact && strings.Contains(name, query))
			}
			for _, p := range byPID {
				key := adapter.group(p, byPID)
				if key.id != "" && matches(key.name) {
					members[key] = append(members[key], p)
				}
			}
			if target == "" || len(members) != 0 {
				return members, nil
			}
			for _, p := range byPID {
				if matches(p.Name) || (p.Executable != "" && matches(filepath.Base(p.Executable))) {
					roots[p.PID] = true
				}
			}
			if len(roots) != 0 {
				break
			}
		}
	}
	for _, p := range byPID {
		var root *Process
		seen := make(map[int32]bool)
		for a := p; a != nil && !seen[a.PID]; a = byPID[a.PPID] {
			seen[a.PID] = true
			if roots[a.PID] {
				root = a // Use the outermost match so overlapping trees count once.
			}
		}
		if root != nil {
			key := groupKey{strconv.Itoa(int(root.PID)), root.Name}
			members[key] = append(members[key], p)
		}
	}
	if len(members) == 0 {
		return nil, fmt.Errorf("no running app or process matches %q", target)
	}
	return members, nil
}

// buildReport assigns every PID to one group before calculating totals.
func buildReport(processes []*Process, target string, interval time.Duration, adapter platformAdapter) (Report, error) {
	byPID := make(map[int32]*Process, len(processes))
	for _, p := range processes {
		p.Children = nil
		byPID[p.PID] = p
	}

	members, err := selectGroups(byPID, target, adapter)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		SampledAt: time.Now().UTC(), SampleSeconds: interval.Seconds(),
		MemoryMetric: memoryMetric, Groups: []Group{},
	}
	for owner, groupProcesses := range members {
		g := Group{Name: owner.name, ProcessCount: len(groupProcesses), Processes: []*Process{}}
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
			if p.MemoryBytes == nil {
				g.MemoryIncomplete = true
			} else {
				if g.MemoryBytes == nil {
					g.MemoryBytes = new(uint64)
				}
				*g.MemoryBytes += *p.MemoryBytes
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
		if a.MemoryBytes != nil {
			left = *a.MemoryBytes
		}
		if b.MemoryBytes != nil {
			right = *b.MemoryBytes
		}
		if order := cmp.Compare(right, left); order != 0 {
			return order
		}
		return strings.Compare(a.Name, b.Name)
	})
	return report, nil
}

func sortProcesses(processes []*Process) {
	slices.SortFunc(processes, func(a, b *Process) int { return cmp.Compare(a.PID, b.PID) })
}
