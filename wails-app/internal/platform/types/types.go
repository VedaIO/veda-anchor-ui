// Package types defines the platform abstraction types and interfaces.
// This package is separate to avoid circular imports between platform and platform/windows.
package types

// WindowInfo represents information about a window on any platform
type WindowInfo struct {
	PID   uint32
	Title string
}

// Platform provides OS-specific functionality
type Platform interface {
	// GetForegroundWindow returns info about the currently focused window.
	// Returns nil if no foreground window is found.
	GetForegroundWindow() *WindowInfo

	// HasVisibleWindow checks if a process has a visible window
	HasVisibleWindow(pid uint32) bool

	// GetIntegrityLevel returns the integrity level of a process (Windows-specific concept)
	// On non-Windows platforms, returns 0 (medium equivalent)
	GetIntegrityLevel(pid uint32) (uint32, error)

	// GetIcon extracts the icon from an executable and returns it as base64 PNG
	GetIcon(exePath string) (string, error)

	// GetProductName returns the product name from executable metadata
	GetProductName(exePath string) (string, error)

	// GetPublisher returns the publisher/signer of an executable
	GetPublisher(exePath string) (string, error)

	// IsMicrosoftSigned checks if an executable is signed by Microsoft
	IsMicrosoftSigned(exePath string) bool
}

// Integrity level constants (Windows values, used as cross-platform reference)
const (
	IntegrityUntrusted = 0x0000
	IntegrityLow       = 0x1000
	IntegrityMedium    = 0x2000
	IntegrityHigh      = 0x3000
	IntegritySystem    = 0x4000
)
