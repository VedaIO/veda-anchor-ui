package main

//go:generate go run github.com/akavel/rsrc -manifest build/procguard.manifest -o build/cache/rsrc.syso

import (
	"database/sql"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"procguard/api"
	"procguard/gui"
	"procguard/internal/daemon"
	"procguard/internal/data"
	"procguard/internal/ipc"
	"procguard/internal/web"
	"strings"

	"time"
)

const (
	// defaultPort is the port used by the web server.
	defaultPort = "58141"
	// internalIPCPort is the port used for internal communication between ProcGuard components.
	internalIPCPort = "58142"
	// chromeExtensionID is the ID of the Chrome extension that communicates with the native messaging host.
	chromeExtensionID = "ilaocldmkhlifnikhinkmiepekpbefoh"
)

// main is the entry point of the application. It determines the execution mode based on command-line arguments.
func main() {
	// Define command-line flags.
	background := flag.Bool("background", false, "Run the application in background (service) mode without a GUI.")
	flag.Parse()

	db, err := data.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	data.NewLogger(db)

	// Determine the application's execution mode.
	// 1. Background (service) mode: Started by autostart, runs silently.
	if *background {
		startBackgroundService(db)
		// Keep the background service running indefinitely.
		select {}
	} else if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "chrome-extension://") {
		runNativeMessagingHost(db)
		return
	} else {
		startGUIApplication(db)
	}
}

// startBackgroundService initializes and runs the core services required for background operation.
// This includes the API server, internal IPC server, and the daemon, without launching a GUI.
func startBackgroundService(db *sql.DB) {
	data.GetLogger().Println("Starting ProcGuard in background service mode...")
	startAPIServer(db)
	startInternalAPIServer(db)
	startDaemonService(db)
}

// runNativeMessagingHost starts the application in native messaging host mode,
// allowing communication with the browser extension.
func runNativeMessagingHost(db *sql.DB) {
	web.Run()
}

// startAPIServer initializes and starts the API server in a new goroutine.
func startAPIServer(db *sql.DB) {
	addr := "127.0.0.1:" + defaultPort
	go api.StartWebServer(addr, registerWebRoutes, db)
}

// startInternalAPIServer initializes and starts the internal IPC server in a new goroutine.
func startInternalAPIServer(db *sql.DB) {
	mux := http.NewServeMux()
	mux.HandleFunc("/log-web-event", ipc.HandleLogWebEvent)
	mux.HandleFunc("/log-web-metadata", ipc.HandleLogWebMetadata)
	mux.HandleFunc("/get-web-blocklist", ipc.HandleGetWebBlocklist)

	addr := "127.0.0.1:" + internalIPCPort
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			data.GetLogger().Fatalf("Failed to start internal API server: %v", err)
		}
	}()
}

// registerWebRoutes sets up the routes for the web server.
func registerWebRoutes(srv *api.Server, r *http.ServeMux) {
	distFS, err := fs.Sub(gui.FrontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("Failed to create sub-filesystem for frontend: %v", err)
	}

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		filePath := strings.TrimPrefix(r.URL.Path, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		content, err := fs.ReadFile(distFS, filePath)
		if err != nil {
			// If the file is not found, serve index.html as a fallback for SPA routing.
			log.Printf("File not found: %s, serving index.html", filePath)
			content, err = fs.ReadFile(distFS, "index.html")
			if err != nil {
				http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			filePath = "index.html" // Set filePath to index.html for content type detection
		}

		contentType := "text/plain"
		if strings.HasSuffix(filePath, ".html") {
			contentType = "text/html; charset=utf-8"
		} else if strings.HasSuffix(filePath, ".css") {
			contentType = "text/css; charset=utf-8"
		} else if strings.HasSuffix(filePath, ".js") {
			contentType = "application/javascript; charset=utf-8"
		} else if strings.HasSuffix(filePath, ".svg") {
			contentType = "image/svg+xml"
		}

		log.Printf("Serving file: %s, content-type: %s", filePath, contentType)

		w.Header().Set("Content-Type", contentType)
		_, err = w.Write(content)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})

	r.HandleFunc("/ping", gui.HandlePing)
}

// startDaemonService initializes and starts the background daemon.
func startDaemonService(db *sql.DB) {
	daemon.StartDaemon(data.GetLogger(), db)
}

// startGUIApplication handles the main startup logic for the GUI application.
func startGUIApplication(db *sql.DB) {
	exePath, err := os.Executable()
	if err != nil {
		data.GetLogger().Printf("Error getting executable path: %v", err)
		// We can continue, but some features might not work.
	}

	// This setup is necessary for the browser extension to communicate with the application.
	if err := web.InstallNativeHost(exePath, chromeExtensionID); err != nil {
		data.GetLogger().Printf("Failed to install native messaging host: %v\n", err)
		// This is not a fatal error, the application can still run without the extension.
	}

	guiAddress := "127.0.0.1:" + defaultPort
	guiUrl := "http://" + guiAddress

	// Check if an instance of the application is already running.
	if isAppRunning(guiUrl) {
		openBrowser(guiUrl)
		return
	}

	// Start the API server and the daemon as goroutines.
	startAPIServer(db)
	startInternalAPIServer(db)
	startDaemonService(db)

	// Give the server a moment to start before opening the browser.
	time.Sleep(1 * time.Second)
	openBrowser(guiUrl)

	// Keep the main GUI application running.
	select {}
}

// isAppRunning checks if another instance of the application is already running by pinging the server.
func isAppRunning(url string) bool {
	resp, err := http.Get(url + "/ping")
	if err != nil {
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			data.GetLogger().Printf("Failed to close response body in isAppRunning: %v", err)
		}
	}()
	return resp.StatusCode == http.StatusOK
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) {
	if err := exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start(); err != nil {
		data.GetLogger().Printf("Error opening browser: %v\n", err)
	}
}
