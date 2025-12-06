//go:build windows

package platform

import (
	"wails-app/internal/platform/windows"
)

func init() {
	SetPlatform(&windows.WindowsPlatform{})
}
