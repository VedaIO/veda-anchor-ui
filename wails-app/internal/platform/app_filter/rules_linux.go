//go:build linux

package app_filter

import (
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// ShouldExclude returns true if the process should be ignored (Stub for non-Windows).
func ShouldExclude(exePath string, proc *process.Process) bool {
	exePathLower := strings.ToLower(exePath)

	// Rule 0: Never track ProcGuard itself
	if strings.Contains(exePathLower, "procguard") {
		return true
	}

	return false
}

// ShouldTrack returns true if the process should be tracked (Stub for non-Windows).
func ShouldTrack(exePath string, proc *process.Process) bool {
	// Simple heuristic for Linux: everything not excluded is tracked
	return true
}
