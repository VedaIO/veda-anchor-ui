# Advanced Logging & Crash Reporting

## Overview
Implement comprehensive logging with ETW (Event Tracing for Windows), crash dump generation, and forensic analysis to diagnose why ProcGuard crashed or was bypassed.

## Current Limitation
**Problem:** Limited crash diagnostics
- If ProcGuard crashes, no stacktrace
- Cannot determine external interference (malware, tampering)
- Logs may be deleted before review
- Custom Essay

## Goals
1. **Crash Dumps:** Generate minidumps on crash for forensic analysis
2. **ETW Logging:** Survive system crashes with circular buffers
3. **Tamper Detection:** Track unauthorized modifications
4. **Forensic Timeline:** Reconstruct events leading to failure

## Technical Approach

### 1. Minidump Generation (Crash Reports)
**WinAPI Required:**
- `MiniDumpWriteDump` - Generate crash dumps
- `SetUnhandledExceptionFilter` - Catch crashes

**CGO Implementation:**
```c
#include <dbghelp.h>
#include <windows.h>

LONG WINAPI UnhandledExceptionHandler(EXCEPTION_POINTERS* exceptionInfo) {
    // Generate crash dump
    HANDLE hFile = CreateFileW(
        L"C:\\ProcGuard\\Logs\\crash.dmp",
        GENERIC_WRITE, 0, NULL, CREATE_ALWAYS,
        FILE_ATTRIBUTE_NORMAL, NULL
    );
    
    if (hFile != INVALID_HANDLE_VALUE) {
        MINIDUMP_EXCEPTION_INFORMATION mdei;
        mdei.ThreadId = GetCurrentThreadId();
        mdei.ExceptionPointers = exceptionInfo;
        mdei.ClientPointers = FALSE;
        
        MINIDUMP_TYPE dumpType = (MINIDUMP_TYPE)(
            MiniDumpWithFullMemory |
            MiniDumpWithHandleData |
            MiniDumpWithThreadInfo
        );
        
        MiniDumpWriteDump(
            GetCurrentProcess(),
            GetCurrentProcessId(),
            hFile,
            dumpType,
            &mdei,
            NULL,
            NULL
        );
        
        CloseHandle(hFile);
    }
    
    // Show error dialog
    MessageBoxW(NULL, 
        L"ProcGuard has crashed. A crash report has been saved to C:\\ProcGuard\\Logs\\crash.dmp\nPlease send this file to support.",
        L"ProcGuard Crash", MB_OK | MB_ICONERROR);
    
    return EXCEPTION_EXECUTE_HANDLER;
}

void SetupCrashHandler() {
    SetUnhandledExceptionFilter(UnhandledExceptionHandler);
}
```

### 2. ETW (Event Tracing) Logging
**WinAPI Required:**
- `EventRegister` - Register ETW provider
- `EventWrite` - Write events to ETW
- `StartTrace` - Create ETW session

**Benefits:**
- Logs survive process crashes
- Circular buffers (last 100 MB preserved)
- Kernel-level logging (tamper-proof)
- Can be read post-crash

**Implementation:**
```c
#include <evntprov.h>

REGHANDLE g_etwHandle;

void InitializeETW() {
    // ProcGuard ETW Provider GUID
    GUID providerId = {0x12345678, 0x1234, 0x1234, {0x12, 0x34, 0x56, 0x78, 0x90, 0xAB, 0xCD, 0xEF}};
    EventRegister(&providerId, NULL, NULL, &g_etwHandle);
}

void LogEventETW(const wchar_t* message, int severity) {
    EVENT_DESCRIPTOR eventDescriptor;
    EventDescCreate(&eventDescriptor, 1, 0, 0, severity, 0, 0, 0);
    
    EVENT_DATA_DESCRIPTOR dataDescriptor;
    EventDataDescCreate(&dataDescriptor, message, 
                        (wcslen(message) + 1) * sizeof(wchar_t));
    
    EventWrite(g_etwHandle, &eventDescriptor, 1, &dataDescriptor);
}
```

**Reading ETW Logs:**
```powershell
# View ProcGuard ETW events
logman query providers ProcGuard
logman create trace ProcGuardTrace -p ProcGuard -o C:\Logs\procguard.etl
logman start ProcGuardTrace

# After crash:
logman stop ProcGuardTrace
tracerpt C:\Logs\procguard.etl -o report.xml
```

### 3. Windows Error Reporting (WER) Integration
**WinAPI Required:**
- `WerRegisterFile` - Register files for crash uploads
- `WerSetFlags` - Configure WER behavior

**Use Case:**
```
When ProcGuard crashes:
  1. Generate minidump
  2. Register minidump with WER
  3. Upload to Microsoft crash servers (with parent consent)
  4. Get crash ID for support tickets
```

**Implementation:**
```c
#include <werapi.h>

void SetupWER() {
    // Register database for crash upload
    WerRegisterFile(L"C:\\ProcGuard\\procguard.db", 
                    WerRegFileTypeOther, WER_FILE_ANONYMOUS_DATA);
    
    // Register log file
    WerRegisterFile(L"C:\\ProcGuard\\Logs\\procguard.log",
                    WerRegFileTypeOther, 0);
    
    // Don't show WER dialog, just upload silently
    WerSetFlags(WER_FAULT_REPORTING_NO_UI);
}
```

### 4. Structured Logging with Metadata

**Log Format (JSON Lines):**
```json
{"timestamp":"2024-01-15T14:32:11Z","level":"INFO","component":"ProcessMonitor","message":"Roblox blocked","pid":1234,"user":"Child","session_id":2}
{"timestamp":"2024-01-15T14:32:15Z","level":"WARN","component":"SelfProtection","message":"Task Manager attempted to terminate ProcGuard","attacker_pid":5678}
{"timestamp":"2024-01-15T14:32:20Z","level":"ERROR","component":"Database","message":"SQLite error: database locked","errno":5}
{"timestamp":"2024-01-15T14:32:21Z","level":"FATAL","component":"Core","message":"Unrecoverable error, shutting down","stacktrace":"..."}
```

**Go Logging Library:**
```go
type StructuredLogger struct {
    file *os.File
}

func (l *StructuredLogger) LogEvent(level, component, message string, metadata map[string]interface{}) {
    entry := map[string]interface{}{
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "level":     level,
        "component": component,
        "message":   message,
    }
    
    for k, v := range metadata {
        entry[k] = v
    }
    
    json.NewEncoder(l.file).Encode(entry)
}
```

### 5. Tamper Detection Logging

**Events to Log:**
- Process termination attempts
- File modification attempts
- Registry changes
- Service stop attempts
- Database corruption

**Example:**
```json
{
  "timestamp": "2024-01-15T14:30:00Z",
  "level": "CRITICAL",
  "event_type": "tamper_attempt",
  "description": "taskkill attempted to terminate ProcGuard",
  "attacker_process": "taskkill.exe",
  "attacker_pid": 9876,
  "attacker_user": "Child",
  "action_taken": "denied",
  "evidence": "C:\\ProcGuard\\Logs\\screenshot_14-30-00.png"
}
```

### 6. Forensic Timeline Reconstruction

**Post-Crash Analysis:**
```sql
-- Reconstruct last 5 minutes before crash
SELECT * FROM logs
WHERE timestamp > datetime('now', '-5 minutes')
ORDER BY timestamp DESC;

-- Find suspicious patterns
SELECT component, COUNT(*) as error_count
FROM logs
WHERE level IN ('ERROR', 'FATAL')
GROUP BY component
ORDER BY error_count DESC;
```

## Database Schema

```sql
CREATE TABLE logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TIMESTAMP NOT NULL,
    level TEXT NOT NULL,  -- DEBUG, INFO, WARN, ERROR, FATAL
    component TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata TEXT,  -- JSON
    stacktrace TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_logs_timestamp ON logs(timestamp);
CREATE INDEX idx_logs_level ON logs(level);

CREATE TABLE crash_reports (
    id INTEGER PRIMARY KEY,
    crash_timestamp TIMESTAMP,
    dump_file_path TEXT,
    exception_code TEXT,
    exception_address TEXT,
    process_id INTEGER,
    thread_id INTEGER,
    etw_trace_path TEXT,  -- Associated ETW trace
    crash_context TEXT,  -- JSON of last 100 log entries
    uploaded_to_wer BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Implementation Phases

### Phase 1: Structured Logging (Pure Go)
**Timeline:** 2-3 days
- [ ] Implement JSON logging
- [ ] Add metadata fields (PID, user, session)
- [ ] Log rotation (max 100 MB)
- [ ] Test log parsing

### Phase 2: Minidump Generation (CGO Required)
**Timeline:** 3-4 days
- [ ] Implement `MiniDumpWriteDump` wrapper
- [ ] Set up exception handler
- [ ] Test with forced crash
- [ ] Verify dump readability (WinDbg)

### Phase 3: ETW Integration (CGO Required)
**Timeline:** 5-7 days
- [ ] Register ETW provider
- [ ] Implement event writing
- [ ] Create ETW session
- [ ] Test circular buffer behavior

### Phase 4: WER Integration (CGO Required)
**Timeline:** 2-3 days
- [ ] Register files with WER
- [ ] Configure silent upload
- [ ] Test crash upload flow
- [ ] Verify uploaded data

### Phase 5: Forensic UI (Svelte)
**Timeline:** 3-4 days
- [ ] Crash report viewer
- [ ] Log search/filter
- [ ] Timeline visualization
- [ ] Export logs to ZIP

## UI Design

### Crash Report Viewer
```svelte
<div class="crash-viewer">
  <h2>Crash Reports</h2>
  {#each crashReports as crash}
    <div class="crash-card">
      <h3>{formatDate(crash.timestamp)}</h3>
      <p><strong>Exception:</strong> {crash.exception_code}</p>
      <p><strong>Address:</strong> {crash.exception_address}</p>
      <button on:click={() => viewDump(crash.dump_file_path)}>
        Open Dump File
      </button>
      <button on:click={() => viewContext(crash.crash_context)}>
        View Log Context
      </button>
      {#if !crash.uploaded_to_wer}
        <button on:click={() => uploadCrash(crash.id)}>
          Upload to Support
        </button>
      {/if}
    </div>
  {/each}
</div>
```

### Log Viewer
```svelte
<div class="log-viewer">
  <input type="text" bind:value={searchQuery} placeholder="Search logs..." />
  <select bind:value={levelFilter}>
    <option value="">All Levels</option>
    <option value="ERROR">Errors Only</option>
    <option value="WARN">Warnings</option>
  </select>
  
  <table>
    <tr>
      <th>Time</th>
      <th>Level</th>
      <th>Component</th>
      <th>Message</th>
    </tr>
    {#each filteredLogs as log}
      <tr class={log.level.toLowerCase()}>
        <td>{formatTime(log.timestamp)}</td>
        <td>{log.level}</td>
        <td>{log.component}</td>
        <td>{log.message}</td>
      </tr>
    {/each}
  </table>
</div>
```

## CGO + Zig Benefits

### Why CGO is Needed:
1. **MiniDumpWriteDump:** Must call `dbghelp.dll`
2. **ETW:** `evntprov.h` requires native calls
3. **Exception Handling:** Structured exception handling (SEH) is C-only

### Zig Advantages:
- Static linking of `dbghelp.lib`
- Clean FFI for exception structures
- Cross-compile crash analysis tools

## Testing Plan

### Test Cases:
1. **Forced Crash:** Trigger access violation, verify minidump generated
2. **ETW Test:** Write 1000 events, crash, verify events readable
3. **WER Upload:** Ensure crash uploaded to Microsoft (with consent)
4. **Log Rotation:** Fill 100 MB, verify old logs deleted
5. **Tamper Log:** Attempt to kill ProcGuard, verify logged

### Success Metrics:
- [ ] 100% crash dumps generated on unhandled exceptions
- [ ] ETW logs survive process termination
- [ ] Crash context includes last 100 log entries
- [ ] < 50 ms logging latency (high-frequency events)

## Privacy & Compliance

### Data Collection:
- Crash dumps may contain sensitive data (passwords in memory)
- **Solution:** Scrub sensitive strings before WER upload
- **Option:** Disable WER upload, local-only crash dumps

### Transparency:
- Clear privacy policy in settings
- Parent can review all crash data before upload
- Option to auto-delete logs after 30 days

## Future Enhancements
- **Cloud Logging:** Optional upload to parent's cloud storage
- **AI Analysis:** ML model detects unusual crash patterns
- **Remote Debugging:** Live attach debugger from support team (with permission)
