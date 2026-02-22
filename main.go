package main

import (
	"context"
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"log"
	"os"
	"path/filepath"
)

// Embed the entire frontend/dist directory into the Go binary
//
//go:embed all:frontend/dist
var assets embed.FS

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func main() {
	// CRITICAL: Log startup for debugging
	cacheDir, _ := os.UserCacheDir()
	logDir := filepath.Join(cacheDir, "VedaAnchor", "logs")
	_ = os.MkdirAll(logDir, 0755)

	logPath := filepath.Join(logDir, "veda-anchor_ui.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
		log.SetOutput(logFile)
	}

	log.Printf("=== ANCHOR UI LAUNCHED === Args: %v", os.Args)

	app := NewApp()

	// Create and run the Wails application
	err := wails.Run(&options.App{
		Title:       "VedaAnchor",
		Width:       1024,
		Height:      768,
		Frameless:   true,
		StartHidden: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,

		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               filepath.Join(os.Getenv("LOCALAPPDATA"), "VedaAnchorUI", "webview"),
		},

		HideWindowOnClose: true,

		// SingleInstanceLock: Ensure only one GUI instance runs
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.vedaio.veda-anchor-ui",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				log.Println("Second GUI instance detected - showing existing window")
				app.ShowWindow()
			},
		},

		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal("Error running Wails app:", err)
	}
}
