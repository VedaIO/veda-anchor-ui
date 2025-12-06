//go:build darwin

package platform

import (
	"wails-app/internal/platform/darwin"
)

func init() {
	SetPlatform(&darwin.DarwinPlatform{})
}
