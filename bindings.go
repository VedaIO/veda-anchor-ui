package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"veda-anchor-ui/internal/ipc"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds the application context and IPC client
type App struct {
	ctx       context.Context
	ipcClient *ipc.Client
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		ipcClient: ipc.NewClient(),
	}
}

// --- Helper ---

func unmarshalResult[T any](raw json.RawMessage) (T, error) {
	var v T
	err := json.Unmarshal(raw, &v)
	return v, err
}

func (a *App) callVoid(method string, params interface{}) error {
	_, err := a.ipcClient.Request(method, params)
	return err
}

func (a *App) callResult(method string, params interface{}) (interface{}, error) {
	res, err := a.ipcClient.Request(method, params)
	if err != nil {
		return nil, err
	}
	var data interface{}
	err = json.Unmarshal(res, &data)
	return data, err
}

// --- Stats ---

func (a *App) GetAppLeaderboard(since, until string) (interface{}, error) {
	return a.callResult("GetAppLeaderboard", map[string]string{"since": since, "until": until})
}

func (a *App) GetScreenTime() (interface{}, error) {
	return a.callResult("GetScreenTime", nil)
}

func (a *App) GetTotalScreenTime() (interface{}, error) {
	return a.callResult("GetTotalScreenTime", nil)
}

func (a *App) GetWebLeaderboard(since, until string) (interface{}, error) {
	return a.callResult("GetWebLeaderboard", map[string]string{"since": since, "until": until})
}

func (a *App) Search(query, since, until string) (interface{}, error) {
	return a.callResult("Search", map[string]string{"query": query, "since": since, "until": until})
}

func (a *App) GetWebLogs(query, since, until string) (interface{}, error) {
	return a.callResult("GetWebLogs", map[string]string{"query": query, "since": since, "until": until})
}

// --- App Blocklist ---

func (a *App) GetAppBlocklist() (interface{}, error) {
	return a.callResult("GetAppBlocklist", nil)
}

func (a *App) BlockApps(names []string) error {
	return a.callVoid("BlockApps", names)
}

func (a *App) UnblockApps(names []string) error {
	return a.callVoid("UnblockApps", names)
}

func (a *App) ClearAppBlocklist() error {
	return a.callVoid("ClearAppBlocklist", nil)
}

func (a *App) SaveAppBlocklist() (interface{}, error) {
	return a.callResult("SaveAppBlocklist", nil)
}

func (a *App) LoadAppBlocklist(content []byte) error {
	return a.callVoid("LoadAppBlocklist", content)
}

// --- Web Blocklist ---

func (a *App) GetWebBlocklist() (interface{}, error) {
	return a.callResult("GetWebBlocklist", nil)
}

func (a *App) AddWebBlocklist(domain string) error {
	return a.callVoid("AddWebBlocklist", domain)
}

func (a *App) RemoveWebBlocklist(domain string) error {
	return a.callVoid("RemoveWebBlocklist", domain)
}

func (a *App) ClearWebBlocklist() error {
	return a.callVoid("ClearWebBlocklist", nil)
}

func (a *App) SaveWebBlocklist() (interface{}, error) {
	return a.callResult("SaveWebBlocklist", nil)
}

func (a *App) LoadWebBlocklist(content []byte) error {
	return a.callVoid("LoadWebBlocklist", content)
}

// --- Auth ---

func (a *App) GetIsAuthenticated() (interface{}, error) {
	return a.callResult("GetIsAuthenticated", nil)
}

func (a *App) Logout() error {
	return a.callVoid("Logout", nil)
}

func (a *App) HasPassword() (interface{}, error) {
	return a.callResult("HasPassword", nil)
}

func (a *App) Login(password string) (interface{}, error) {
	return a.callResult("Login", map[string]string{"password": password})
}

func (a *App) SetPassword(password string) error {
	return a.callVoid("SetPassword", map[string]string{"password": password})
}

// --- System ---

func (a *App) Shutdown() error {
	return a.callVoid("Shutdown", nil)
}

func (a *App) Uninstall(password string) error {
	return a.callVoid("Uninstall", map[string]string{"password": password})
}

func (a *App) GetAutostartStatus() (interface{}, error) {
	return a.callResult("GetAutostartStatus", nil)
}

func (a *App) EnableAutostart() error {
	return a.callVoid("EnableAutostart", nil)
}

func (a *App) DisableAutostart() error {
	return a.callVoid("DisableAutostart", nil)
}

func (a *App) ClearAppHistory(password string) error {
	return a.callVoid("ClearAppHistory", map[string]string{"password": password})
}

func (a *App) ClearWebHistory(password string) error {
	return a.callVoid("ClearWebHistory", map[string]string{"password": password})
}

// --- Local Methods (UI-side only) ---

func (a *App) CheckChromeExtension() bool {
	progData := os.Getenv("ProgramData")
	if progData == "" {
		progData = `C:\ProgramData`
	}
	heartbeatPath := filepath.Join(progData, "VedaAnchor", "extension_heartbeat")
	content, err := os.ReadFile(heartbeatPath)
	if err != nil {
		return false
	}
	var lastPing int64
	if _, err := fmt.Sscanf(string(content), "%d", &lastPing); err != nil {
		return false
	}
	return time.Since(time.Unix(lastPing, 0)) < 10*time.Second
}

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

func (a *App) ShowWindow() {
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.Show(a.ctx)
}
