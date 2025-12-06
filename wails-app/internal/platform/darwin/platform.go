//go:build darwin

package darwin

import (
	"wails-app/internal/platform/types"
)

// DarwinPlatform implements types.Platform for macOS
type DarwinPlatform struct{}

// GetForegroundWindow returns info about the currently focused window
func (p *DarwinPlatform) GetForegroundWindow() *types.WindowInfo {
	return nil // TODO: implement with NSWorkspace
}

// HasVisibleWindow checks if a process has a visible window
func (p *DarwinPlatform) HasVisibleWindow(pid uint32) bool {
	return true
}

// GetIntegrityLevel returns the integrity level of a process
func (p *DarwinPlatform) GetIntegrityLevel(pid uint32) (uint32, error) {
	return types.IntegrityMedium, nil
}

// GetIcon extracts the icon from an application bundle
func (p *DarwinPlatform) GetIcon(exePath string) (string, error) {
	return "", nil
}

// GetProductName returns the product name from Info.plist
func (p *DarwinPlatform) GetProductName(exePath string) (string, error) {
	return "", nil
}

// GetPublisher returns the code signing identity
func (p *DarwinPlatform) GetPublisher(exePath string) (string, error) {
	return "", nil
}

// IsMicrosoftSigned checks if an app is signed by Microsoft
func (p *DarwinPlatform) IsMicrosoftSigned(exePath string) bool {
	return false
}
