//go:build darwin

package api

import (
	"path/filepath"
	"strings"
)

// getAppDetails handles retrieving the application name on macOS
func (srv *Server) getAppDetails(exePath string) (string, string) {
	// On macOS, applications are usually .app bundles or binaries
	commercialName := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))
	return commercialName, ""
}
