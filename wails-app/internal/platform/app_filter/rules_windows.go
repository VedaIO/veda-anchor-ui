//go:build windows

package app_filter

import (
	"strings"
	"wails-app/internal/platform/executable"
	"wails-app/internal/platform/integrity"
	"wails-app/internal/platform/window"

	"github.com/shirou/gopsutil/v3/process"
)

// ShouldExclude returns true if the process is a Windows system component, conhost.exe, or ProcGuard itself.
func ShouldExclude(exePath string, proc *process.Process) bool {
	exePathLower := strings.ToLower(exePath)

	// Never track ProcGuard itself
	if strings.Contains(exePathLower, "procguard.exe") {
		return true
	}

	// Skip conhost.exe
	if strings.HasSuffix(exePathLower, "conhost.exe") {
		return true
	}

	// Skip if in System32/SysWOW64 (Windows system processes)
	if strings.Contains(exePathLower, "\\windows\\system32\\") ||
		strings.Contains(exePathLower, "\\windows\\syswow64\\") {
		return true
	}

	// Skip processes with "Microsoft® Windows® Operating System" product name
	productName, err := executable.GetProductName(exePath)
	if err == nil && strings.Contains(productName, "Microsoft® Windows® Operating System") {
		return true
	}

	// Skip system integrity level processes (system services)
	if proc != nil {
		il, err := integrity.GetProcessLevel(uint32(proc.Pid))
		if err == nil && il >= integrity.SystemRID {
			return true
		}
	}

	return false
}

// ShouldTrack returns true if the process is a user application that should be monitored.
func ShouldTrack(exePath string, proc *process.Process) bool {
	if proc == nil {
		return false
	}

	name, err := proc.Name()
	if err != nil {
		return false
	}
	nameLower := strings.ToLower(name)

	// Log cmd.exe and powershell.exe ONLY if launched by explorer.exe
	if nameLower == "cmd.exe" || nameLower == "powershell.exe" || nameLower == "pwsh.exe" {
		parent, err := proc.Parent()
		if err == nil {
			parentName, err := parent.Name()
			if err == nil && strings.EqualFold(parentName, "explorer.exe") {
				return true
			}
		}
		return false
	}

	// Must have visible window (user interaction indicator)
	if !window.HasVisibleWindow(uint32(proc.Pid)) {
		return false
	}

	// Prefer processes launched by explorer.exe (Start menu, desktop)
	parent, err := proc.Parent()
	if err == nil {
		parentName, err := parent.Name()
		if err == nil && strings.EqualFold(parentName, "explorer.exe") {
			return true
		}
	}

	// If it has a visible window and isn't a shell, we usually want to track it
	return true
}
