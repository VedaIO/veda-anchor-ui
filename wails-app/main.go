package main

import (
	"context"
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"log"
	"os"
	"strings"
	"wails-app/api"
	"wails-app/internal/daemon"
	"wails-app/internal/data"
	"wails-app/internal/web"
)

// Embed the entire frontend/dist directory into the Go binary
// This allows the app to be distributed as a single executable
//
//go:embed all:frontend/dist
var assets embed.FS

// startup is called when the Wails app starts
// The context is saved so we can call runtime methods (WindowShow, etc.) later
//
// Responsibilities:
//  1. Save the Wails runtime context for later use
//  2. Initialize the database
//  3. Initialize the logger
//  4. Create the API server
//  5. Start the background daemon for process/web monitoring
//  6. Start the native messaging host for Chrome extension communication
func (a *App) startup(ctx context.Context) {
	// Save context - CRITICAL for calling ShowWindow() and other runtime methods
	a.ctx = ctx

	// Initialize database connection
	db, err := data.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize logger with database
	data.NewLogger(db)

	// Create API server with database connection
	a.Server = api.NewServer(db)

	// Start the background daemon that monitors processes and web activity
	// This runs independently of the GUI - continues even when window is hidden
	daemon.StartDaemon(a.Server.Logger, db)

	// Ensure Native Messaging Host is registered
	// This creates the registry key and manifest file so Chrome can find us
	// We do this on every startup to ensure the config is correct
	if err := web.RegisterExtension("hkanepohpflociaodcicmmfbdaohpceo"); err != nil {
		log.Printf("Failed to register Store extension: %v", err)
	}
	if err := web.RegisterExtension("gpaafgcbiejjpfdgmjglehboafdicdjb"); err != nil {
		log.Printf("Failed to register Dev extension: %v", err)
	}

	// Start the native messaging host
	// This listens for messages from the Chrome extension via Stdin
	// CRITICAL: This must run in a goroutine so it doesn't block startup
	go web.Run()
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Check if this is a native messaging launch (first instance)
	// Chrome passes the extension origin as an argument: chrome-extension://<id>/
	isNativeMessaging := false
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "chrome-extension://") {
			isNativeMessaging = true
			break
		}
	}
	
	// Fallback to WD check
	if !isNativeMessaging {
		wd, _ := os.Getwd()
		if wd != "" {
			wdLower := strings.ToLower(wd)
			if strings.Contains(wdLower, "chrome") || strings.Contains(wdLower, "google") {
				isNativeMessaging = true
			}
		}
	}

	if isNativeMessaging {
		log.Println("First instance launched by Chrome (Native Messaging)")
		app.IsNativeMessagingActive = true
	}

	// Create and run the Wails application with configuration
	err := wails.Run(&options.App{
		Title:  "ProcGuard",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,

		// HideWindowOnClose: Hide window instead of closing application when X is clicked
		// Why: Allows daemon to keep running in background while window is hidden
		// User can reopen by double-clicking executable (SingleInstanceLock handles this)
		HideWindowOnClose: true,

		// SingleInstanceLock: Prevent multiple instances of the application
		//
		// Without this, running the executable multiple times creates multiple processes.
		// Combined with HideWindowOnClose=true, this would cause hidden processes to accumulate.
		//
		// With SingleInstanceLock, only one process can run at a time.
		// Subsequent launches trigger the OnSecondInstanceLaunch callback instead.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.procguard.wails-app",

			// OnSecondInstanceLaunch: Callback when executable is run while already running
			//
			// Smart behavior: Distinguish between user launch vs background operation
			//
			// Problem to solve:
			//   - Chrome extension reconnects via native messaging frequently (every 5s)
			//   - This causes the executable to launch (native messaging host)
			//   - But user also needs a way to reopen the hidden window by double-clicking
			//
			// Solution: Check working directory of the launch
			//   - If launched from Chrome's directory → Native messaging, stay hidden
			//   - Otherwise → User intentional launch, show window
			//
			// How to reopen: Just double-click the executable (normal user behavior)
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				// Check if launched by Chrome (Native Messaging)
				// Chrome passes the extension origin as an argument: chrome-extension://<id>/
				isNativeMessaging := false
				
				// Method 1: Check arguments
				for _, arg := range data.Args {
					if strings.HasPrefix(arg, "chrome-extension://") {
						isNativeMessaging = true
						break
					}
				}
				
				// Method 2: Fallback to Working Directory check (just in case)
				if !isNativeMessaging {
					wd := data.WorkingDirectory
					if wd != "" {
						wdLower := strings.ToLower(wd)
						if strings.Contains(wdLower, "chrome") || strings.Contains(wdLower, "google") {
							isNativeMessaging = true
						}
					}
				}
				
				if isNativeMessaging {
					log.Println("Second instance from native messaging - staying hidden")
					
					// Mark native messaging as active
					app.IsNativeMessagingActive = true
					
					// Notify frontend that extension is connected
					// This updates the UI immediately without needing a reload
					if app.ctx != nil {
						wailsruntime.EventsEmit(app.ctx, "extension_connected", true)
					}
				} else {
					log.Println("Second instance from user - showing window")
					// User double-clicked the exe - show the window
					app.ShowWindow()
				}
			},
		},

		// Bind the app struct to make its methods available to frontend JS
		// Frontend can call these via window.go.main.App.MethodName()
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
