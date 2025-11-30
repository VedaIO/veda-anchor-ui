package web

import (
	"os"
	"path/filepath"
	"runtime"
)

// CheckChromeExtension checks if the ProcGuard Chrome extension is installed
// by looking for it in the Chrome extensions directory on the filesystem
//
// Background: In the original browser-based app, we could use chrome.runtime.sendMessage
//             to check if the extension was installed. In Wails WebView, those APIs don't work
//             because the WebView is isolated from the browser's extension environment.
//
// Solution: Check if the extension directory exists on disk. Chrome stores extensions in:
//   - Windows: %LOCALAPPDATA%\Google\Chrome\User Data\Default\Extensions\{extensionID}
//   - macOS:   ~/Library/Application Support/Google/Chrome/Default/Extensions/{extensionID}
//   - Linux:   ~/.config/google-chrome/Default/Extensions/{extensionID}
//
// Limitations: This checks the default Chrome profile only. Extensions in other profiles
//              or browser variants (Chromium, Brave, etc.) won't be detected.
//
// Returns: true if extension directory exists, false otherwise
func CheckChromeExtension() bool {
	// Extension ID from Chrome Web Store
	// https://chromewebstore.google.com/detail/procguard-web-monitor/hkanepohpflociaodcicmmfbdaohpceo
	var extensionID = "hkanepohpflociaodcicmmfbdaohpceo"
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Can't get home directory - assume extension not installed
		return false
	}

	// Build the path to the extension directory based on OS
	// Check for Store Extension
	if checkPath(extensionID, homeDir) {
		return true
	}

	// Check for Dev Extension
	// ID: gpaafgcbiejjpfdgmjglehboafdicdjb
	if checkPath("gpaafgcbiejjpfdgmjglehboafdicdjb", homeDir) {
		return true
	}

	return false
}

func checkPath(id, homeDir string) bool {
	var extensionPath string
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		extensionPath = filepath.Join(localAppData, "Google", "Chrome", "User Data", "Default", "Extensions", id)
	case "darwin":
		extensionPath = filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "Default", "Extensions", id)
	case "linux":
		extensionPath = filepath.Join(homeDir, ".config", "google-chrome", "Default", "Extensions", id)
	default:
		return false
	}

	if _, err := os.Stat(extensionPath); err == nil {
		return true
	}
	return false
}
