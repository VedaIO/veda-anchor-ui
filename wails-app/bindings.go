package main

import (
	"context"
	"os/exec"
	"runtime"
	"wails-app/api"
	"wails-app/internal/web"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds the application context and server instance
// ctx: Wails runtime context - used to call runtime methods like WindowShow, WindowUnminimise
// *api.Server: Embedded server instance that handles all business logic
type App struct {
	ctx context.Context
	*api.Server
	
	// IsNativeMessagingActive indicates if the app was launched by Chrome native messaging
	// This is used to detect if the extension is installed/active, even if unpacked (dev mode)
	IsNativeMessagingActive bool
}

// NewApp creates a new App application struct
// This is called from main() to initialize the application
func NewApp() *App {
	return &App{}
}

// CheckChromeExtension checks if the ProcGuard Chrome extension is installed
// by looking for it in Chrome's extensions directory on the filesystem
// OR by checking if native messaging is active (which means extension launched us)
//
// Why this exists: In a Wails WebView, we can't use chrome.runtime APIs directly
// Solution: Check if the extension directory exists on disk
//
// Returns: true if extension directory found OR native messaging is active
func (a *App) CheckChromeExtension() bool {
	// If native messaging is active, the extension is definitely installed and working!
	// This handles the "dev mode" / unpacked extension case where ID might match
	// but path is different, or if we just want to be sure it's actually connected.
	if a.IsNativeMessagingActive {
		return true
	}

	return web.CheckChromeExtension()
}

// OpenBrowser opens a URL in the user's default system browser
//
// Why this exists: window.open() opens URLs inside the Wails WebView, not external browser
// Problem this fixes: Clicking "Install Extension" was trying to open Chrome Web Store in WebView
// Solution: Use OS-specific commands to open external browser
//
// Platform support:
//   - Windows: uses 'cmd /c start'
//   - macOS: uses 'open'
//   - Linux: uses 'xdg-open'
//
// Returns: error if command fails to start, nil on success
func (a *App) OpenBrowser(url string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	}
	
	return cmd.Start()
}

// ShowWindow brings the application window to the foreground
// Unminimizes the window if it's minimized, then makes it visible
//
// IMPORTANT: Only call this when user explicitly wants to see the window!
// DO NOT call this from polling/background operations or it will interrupt the user
//
// Used by: OnSecondInstanceLaunch callback - when user double-clicks exe while app is running
// Context: With HideWindowOnClose=true, closing the window hides it but keeps daemon running
//          When user runs exe again, SingleInstanceLock prevents new process and calls this instead
func (a *App) ShowWindow() {
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.Show(a.ctx)
}
