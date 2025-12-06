//go:build windows

package windows

/*
#include <windows.h>
#include <stdint.h>

typedef struct {
    uint32_t pid;
    wchar_t title[256];
} ActiveWindowInfo;

int GetActiveWindowInfo(ActiveWindowInfo* info) {
    if (info == NULL) {
        return -1;
    }

    info->pid = 0;
    info->title[0] = L'\0';

    HWND hwnd = GetForegroundWindow();
    if (hwnd == NULL) {
        return -1;
    }

    DWORD pid = 0;
    GetWindowThreadProcessId(hwnd, &pid);
    if (pid == 0) {
        return -1;
    }
    info->pid = (uint32_t)pid;

    GetWindowTextW(hwnd, info->title, 256);

    return 0;
}
*/
import "C"

import (
	"wails-app/internal/platform/types"
)

// WindowsPlatform implements types.Platform for Windows
type WindowsPlatform struct{}

// GetForegroundWindow returns info about the currently focused window using CGO
func (p *WindowsPlatform) GetForegroundWindow() *types.WindowInfo {
	var cInfo C.ActiveWindowInfo
	result := C.GetActiveWindowInfo(&cInfo)
	if result != 0 {
		return nil
	}

	title := wcharToString(cInfo.title[:])

	return &types.WindowInfo{
		PID:   uint32(cInfo.pid),
		Title: title,
	}
}

func wcharToString(wchars []C.wchar_t) string {
	length := 0
	for i, c := range wchars {
		if c == 0 {
			length = i
			break
		}
	}
	if length == 0 {
		return ""
	}

	utf16 := make([]uint16, length)
	for i := 0; i < length; i++ {
		utf16[i] = uint16(wchars[i])
	}

	runes := make([]rune, len(utf16))
	for i, v := range utf16 {
		runes[i] = rune(v)
	}
	return string(runes)
}

// HasVisibleWindow checks if a process has a visible window
func (p *WindowsPlatform) HasVisibleWindow(pid uint32) bool {
	return true // TODO: migrate from app/process.go
}

// GetIntegrityLevel returns the integrity level of a process
func (p *WindowsPlatform) GetIntegrityLevel(pid uint32) (uint32, error) {
	return types.IntegrityMedium, nil // TODO: migrate from app/integrity.go
}

// GetIcon extracts the icon from an executable
func (p *WindowsPlatform) GetIcon(exePath string) (string, error) {
	return "", nil // TODO: migrate from app/icon.go
}

// GetProductName returns the product name from executable metadata
func (p *WindowsPlatform) GetProductName(exePath string) (string, error) {
	return "", nil // TODO: migrate from app/process.go
}

// GetPublisher returns the publisher/signer of an executable
func (p *WindowsPlatform) GetPublisher(exePath string) (string, error) {
	return "", nil // TODO: migrate from app/process.go
}

// IsMicrosoftSigned checks if an executable is signed by Microsoft
func (p *WindowsPlatform) IsMicrosoftSigned(exePath string) bool {
	return false // TODO: migrate from app/process.go
}
