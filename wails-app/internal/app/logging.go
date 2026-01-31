package app

import (
	"database/sql"
	"strings"
	"sync"
	"time"
	"wails-app/internal/data/logger"
	"wails-app/internal/data/write"
	"wails-app/internal/platform/app_filter"

	"github.com/shirou/gopsutil/v3/process"
)

const processCheckInterval = 2 * time.Second

// loggedApps tracks which applications have already been logged (deduplication)
// Key is lowercase process name (e.g., "chrome.exe")
var loggedApps = make(map[string]bool)
var loggedAppsMu sync.Mutex

var resetLoggerCh = make(chan struct{}, 1)

// ResetLoggedApps clears the in-memory cache of logged applications.
// This allows applications that were previously logged to be logged again
// after a history clear.
func ResetLoggedApps() {
	resetLoggerCh <- struct{}{}
}

// StartProcessEventLogger starts a long-running goroutine that monitors process creation and termination events.
func StartProcessEventLogger(appLogger logger.Logger, db *sql.DB) {
	go func() {
		runningProcs := make(map[int32]bool)
		initializeRunningProcs(runningProcs, db)

		ticker := time.NewTicker(processCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				procs, err := process.Processes()
				if err != nil {
					appLogger.Printf("Failed to get processes: %v", err)
					continue
				}

				currentProcs := make(map[int32]bool)
				for _, p := range procs {
					currentProcs[p.Pid] = true
				}

				logEndedProcesses(appLogger, db, runningProcs, currentProcs)
				logNewProcesses(appLogger, db, runningProcs, procs)
			case <-resetLoggerCh:
				appLogger.Printf("[Logger] Reset signal received. Clearing in-memory state.")
				loggedAppsMu.Lock()
				loggedApps = make(map[string]bool)
				loggedAppsMu.Unlock()

				// Clear runningProcs completely.
				// This ensures that even currently running apps will be re-detected as "new"
				// in the next ticker cycle and logged to the cleared database.
				runningProcs = make(map[int32]bool)
			}
		}
	}()
}

func logEndedProcesses(appLogger logger.Logger, db *sql.DB, runningProcs, currentProcs map[int32]bool) {
	for pid := range runningProcs {
		if !currentProcs[pid] {
			write.EnqueueWrite("UPDATE app_events SET end_time = ? WHERE pid = ? AND end_time IS NULL", time.Now().Unix(), pid)
			delete(runningProcs, pid)
		}
	}
}

func logNewProcesses(appLogger logger.Logger, db *sql.DB, runningProcs map[int32]bool, procs []*process.Process) {
	for _, p := range procs {
		if !runningProcs[p.Pid] {
			if shouldLogProcess(p) {
				name, _ := p.Name()

				parent, _ := p.Parent()
				parentName := ""
				if parent != nil {
					parentName, _ = parent.Name()
				}

				exePath, err := p.Exe()
				if err != nil {
					appLogger.Printf("Failed to get exe path for %s (pid %d): %v", name, p.Pid, err)
				}
				write.EnqueueWrite("INSERT INTO app_events (process_name, pid, parent_process_name, exe_path, start_time) VALUES (?, ?, ?, ?, ?)",
					name, p.Pid, parentName, exePath, time.Now().Unix())
				runningProcs[p.Pid] = true
			}
		}
	}
}

func initializeRunningProcs(runningProcs map[int32]bool, db *sql.DB) {
	rows, err := db.Query("SELECT pid FROM app_events WHERE end_time IS NULL")
	if err != nil {
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.GetLogger().Printf("Failed to close rows: %v", err)
		}
	}()

	for rows.Next() {
		var pid int32
		if err := rows.Scan(&pid); err == nil {
			if exists, _ := process.PidExists(pid); exists {
				runningProcs[pid] = true
			} else {
				write.EnqueueWrite("UPDATE app_events SET end_time = ? WHERE pid = ? AND end_time IS NULL", time.Now().Unix(), pid)
			}
		}
	}
}

func shouldLogProcess(p *process.Process) bool {
	name, err := p.Name()
	if err != nil || name == "" {
		return false
	}

	nameLower := strings.ToLower(name)

	// Deduplication - Only log first instance of each application
	loggedAppsMu.Lock()
	if loggedApps[nameLower] {
		loggedAppsMu.Unlock()
		return false
	}
	loggedAppsMu.Unlock()

	// Must be a trackable user application
	exePath, err := p.Exe()
	if err != nil {
		return false
	}

	if app_filter.ShouldExclude(exePath, p) {
		return false
	}

	if !app_filter.ShouldTrack(exePath, p) {
		return false
	}

	// Default: Log it (likely a user application)
	loggedAppsMu.Lock()
	loggedApps[nameLower] = true
	loggedAppsMu.Unlock()
	return true
}
