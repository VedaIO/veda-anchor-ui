// Package platform provides OS-specific functionality abstracted behind interfaces.
// This allows the app and web domains to use platform features without
// knowing the underlying OS implementation details.
package platform

import (
	"wails-app/internal/platform/types"
)

// Re-export types for convenient access
type WindowInfo = types.WindowInfo
type Platform = types.Platform

// Re-export constants
const (
	IntegrityUntrusted = types.IntegrityUntrusted
	IntegrityLow       = types.IntegrityLow
	IntegrityMedium    = types.IntegrityMedium
	IntegrityHigh      = types.IntegrityHigh
	IntegritySystem    = types.IntegritySystem
)

// current holds the platform implementation for the current OS
// Set by init() in init_windows.go or init_darwin.go
var current types.Platform

// Current returns the platform implementation for the current OS
func Current() types.Platform {
	return current
}

// SetPlatform sets the current platform implementation (called from OS-specific init)
func SetPlatform(p types.Platform) {
	current = p
}
