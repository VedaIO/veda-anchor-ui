# Accurate Screen Time Tracking

## Overview
Implement precise screen time tracking that measures active window usage, not just process existence. Track when applications are in foreground vs background, idle time, and multi-monitor scenarios.

## Current Limitation
**Problem:** Current system tracks process lifetime, not actual usage
```
chrome.exe running for 8 hours != 8 hours of active use
  - User might be using other apps
  - Browser might be minimized
  - User might be AFK
```

## Goals
1. **Foreground Tracking**: Only count time when app is active window
2. **Idle Detection**: Pause timer if user idle >5 minutes
3. **Multi-Monitor Support**: Track active monitor per window
4. **Tab-Level Tracking**: For browsers, track which website is active
5. **Focus Time**: Detect if window has keyboard/mouse focus

## Technical Approach

### 1. Foreground Window Tracking
**WinAPI Required:**
- `GetForegroundWindow` - Get currently focused window
- `GetWindowThreadProcessId` - Map window to process
- `GetWindowTextW` - Get window title for logging

**Architecture:**
```go
type ActiveWindowMonitor struct {
    lastForegroundPID uint32
    lastCheckTime     time.Time
    activeTimers      map[uint32]*TimeTracker
}

func (m *ActiveWindowMonitor) Poll() {
    hwnd := GetForegroundWindow()
    var pid uint32
    GetWindowThreadProcessId(hwnd, &pid)
    
    if pid != m.lastForegroundPID {
        // Window changed
        m.StopTimer(m.lastForegroundPID)
        m.StartTimer(pid)
        m.lastForegroundPID = pid
    }
    
    m.lastCheckTime = time.Now()
}
```.

**CGO Implementation:**
```c
#include <windows.h>

typedef struct {
    DWORD pid;
    WCHAR title[256];
    BOOL isFullscreen;
} ActiveWindowInfo;

ActiveWindowInfo GetActiveWindowInfo() {
    ActiveWindowInfo info = {0};
    HWND hwnd = GetForegroundWindow();
    
    if (hwnd) {
        GetWindowThreadProcessId(hwnd, &info.pid);
        GetWindowTextW(hwnd, info.title, 256);
        
        // Check if fullscreen (likely gaming/video)
        RECT rect;
        GetWindowRect(hwnd, &rect);
        int screenWidth = GetSystemMetrics(SM_CXSCREEN);
        int screenHeight = GetSystemMetrics(SM_CYSCREEN);
        
        if (rect.right - rect.left == screenWidth && 
            rect.bottom - rect.top == screenHeight) {
            info.isFullscreen = TRUE;
        }
    }
    
    return info;
}
```

### 2. User Idle Detection
**WinAPI Required:**
- `GetLastInputInfo` - Get timestamp of last keyboard/mouse activity

**Idle Logic:**
```c
DWORD GetIdleTimeMs() {
    LASTINPUTINFO lii = {0};
    lii.cbSize = sizeof(LASTINPUTINFO);
    
    if (GetLastInputInfo(&lii)) {
        DWORD currentTime = GetTickCount();
        return currentTime - lii.dwTime;
    }
    return 0;
}
```

**Usage:**
```
If idle > 5 minutes:
  - Pause all active timers
  - Resume on keyboard/mouse activity
```

### 3. Window State Tracking
**WinAPI Required:**
- `IsIconic` - Check if window minimized
- `IsWindowVisible` - Check if window visible
- `IsZoomed` - Check if window maximized

**Enhanced Tracking:**
```c
typedef enum {
    WS_ACTIVE_FOREGROUND,
    WS_ACTIVE_BACKGROUND,
    WS_MINIMIZED,
    WS_HIDDEN,
    WS_FULLSCREEN
} WindowState;

WindowState GetWindowState(HWND hwnd) {
    if (!IsWindowVisible(hwnd)) return WS_HIDDEN;
    if (IsIconic(hwnd)) return WS_MINIMIZED;
    
    HWND fgWnd = GetForegroundWindow();
    if (hwnd == fgWnd) {
        // Check fullscreen
        RECT rect;
        GetWindowRect(hwnd, &rect);
        if (/* is fullscreen */) return WS_FULLSCREEN;
        return WS_ACTIVE_FOREGROUND;
    }
    
    return WS_ACTIVE_BACKGROUND;
}
```

### 4. Multi-Monitor Support
**WinAPI Required:**
- `MonitorFromWindow` - Get monitor handle for window
- `GetMonitorInfoW` - Get monitor details

**Use Case:**
```
User has dual monitors:
  - Monitor 1: Game (fullscreen)
  - Monitor 2: Browser (background)

ProcGuard tracks:
  - Game: Active (foreground on Monitor 1)
  - Browser: Background (not counted unless explicitly enabled)
```

### 5. Browser Tab Tracking (Advanced)
**WinAPI Required:**
- `UI Automation API` - Read browser accessibility tree
- `FindWindowExW` + `SendMessageW` - Query browser internals

**Approach:**
```c
// For Chrome/Edge (Chromium-based)
#include <uiautomation.h>

HRESULT GetActiveTabUrl(HWND browserWnd, WCHAR* url, int bufSize) {
    IUIAutomation* pAuto;
    CoCreateInstance(&CLSID_CUIAutomation, NULL, CLSCTX_INPROC_SERVER, 
                      &IID_IUIAutomation, (void**)&pAuto);
    
    IUIAutomationElement* pRoot;
    pAuto->ElementFromHandle(browserWnd, &pRoot);
    
    // Navigate UI tree to find address bar
    IUIAutomationCondition* pCondition;
    VARIANT varRole;
    varRole.vt = VT_I4;
    varRole.lVal = UIA_EditControlTypeId;
    pAuto->CreatePropertyCondition(UIA_ControlTypePropertyId, varRole, &pCondition);
    
    IUIAutomationElement* pAddressBar;
    pRoot->FindFirst(TreeScope_Descendants, pCondition, &pAddressBar);
    
    // Get URL value
    VARIANT varValue;
    pAddressBar->GetCurrentPropertyValue(UIA_ValueValuePropertyId, &varValue);
    wcsncpy(url, varValue.bstrVal, bufSize);
    
    // Cleanup...
    return S_OK;
}
```

**Storage:**
```go
type BrowserSession struct {
    ProcessID   uint32
    WindowTitle string
    ActiveURL   string  // "youtube.com/watch?v=..."
    Category    string  // "entertainment", "education", "work"
    StartTime   time.Time
    Duration    time.Duration
}
```

### 6. Polling Strategy
**Current:** Poll every 1 second (high CPU)
**Optimized:** Event-driven + adaptive polling

**WinAPI Required:**
- `SetWinEventHook` - Register for window focus change events

**Implementation:**
```c
HWINEVENTHOOK g_hook;

void CALLBACK WinEventProc(HWINEVENTHOOK hook, DWORD event, HWND hwnd, 
                           LONG idObject, LONG idChild, 
                           DWORD dwEventThread, DWORD dwmsEventTime) {
    if (event == EVENT_SYSTEM_FOREGROUND) {
        // Window focus changed
        NotifyGoCallback(hwnd);
    }
}

void StartForegroundMonitoring() {
    g_hook = SetWinEventHook(
        EVENT_SYSTEM_FOREGROUND, EVENT_SYSTEM_FOREGROUND,
        NULL, WinEventProc, 0, 0, WINEVENT_OUTOFCONTEXT
    );
}
```

**Benefits:**
- Zero CPU when no window changes
- Instant detection (<10ms latency)
- Battery-friendly for laptops

## Database Schema

### Enhanced Time Tracking Table
```sql
CREATE TABLE accurate_screen_time (
    id INTEGER PRIMARY KEY,
    process_id INTEGER,
    executable_path TEXT,
    window_title TEXT,
    active_url TEXT,  -- For browsers
    session_start TIMESTAMP,
    session_end TIMESTAMP,
    duration_seconds INTEGER,
    is_foreground BOOLEAN,
    is_fullscreen BOOLEAN,
    idle_time_seconds INTEGER,
    monitor_id INTEGER,
    FOREIGN KEY (process_id) REFERENCES processes(id)
);
```

### Aggregated View
```sql
CREATE VIEW daily_screen_time AS
SELECT 
    DATE(session_start) as date,
    executable_path,
    SUM(duration_seconds - idle_time_seconds) as active_seconds,
    COUNT(*) as session_count,
    SUM(CASE WHEN is_fullscreen THEN duration_seconds ELSE 0 END) as fullscreen_seconds
FROM accurate_screen_time
WHERE is_foreground = TRUE
GROUP BY DATE(session_start), executable_path;
```

## Implementation Phases

### Phase 1: Foreground Tracking (CGO Required)
**Timeline:** 3-4 days
- [ ] Implement `GetForegroundWindow` polling
- [ ] Map windows to processes
- [ ] Track foreground time per process
- [ ] Test with multi-window apps (browsers)

### Phase 2: Idle Detection (CGO Required)
**Timeline:** 1-2 days
- [ ] Implement `GetLastInputInfo` wrapper
- [ ] Pause timers on idle
- [ ] Resume on activity
- [ ] Test idle threshold tuning (5min vs 10min)

### Phase 3: Window State Tracking (CGO Required)
**Timeline:** 2-3 days
- [ ] Detect minimized windows
- [ ] Detect fullscreen windows (gaming indicator)
- [ ] Track window visibility
- [ ] Multi-monitor detection

### Phase 4: Event-Driven Monitoring (CGO Required)
**Timeline:** 4-5 days
- [ ] Implement `SetWinEventHook`
- [ ] Replace polling with event callbacks
- [ ] Optimize callback -> Go channel pipeline
- [ ] Test CPU usage (<1% idle)

### Phase 5: Browser Tab Tracking (CGO + COM)
**Timeline:** 7-10 days
- [ ] UI Automation API integration
- [ ] Extract active URL from Chrome/Edge
- [ ] Category classification (YouTube vs Google Docs)
- [ ] Privacy controls (disable URL tracking option)

## UI Integration

### Dashboard Updates
```svelte
<div class="screen-time-card">
  <h3>Today's Screen Time</h3>
  <div class="app-usage">
    <div class="app-row">
      <img src="chrome-icon.png" />
      <div class="details">
        <span class="name">Google Chrome</span>
        <span class="time">2h 34m active</span>
        <span class="subtitle">YouTube: 1h 12m, Docs: 45m</span>
      </div>
    </div>
  </div>
</div>
```

### Real-Time Indicator
```
Current Activity: Roblox (Fullscreen) - 23 minutes remaining
[Pause Timer] [Add 30 Minutes]
```

## Privacy Considerations
- **URL Tracking Toggle**: Parent can disable browser tab tracking
- **Screenshot Opt-In**: Optional periodic screenshots (disabled by default)
- **Local Storage**: All data stays on device (never uploaded)

## CGO + Zig Benefits

### Why CGO is Needed:
1. **Event Hooks**: `SetWinEventHook` requires callback function (CGO only)
2. **UI Automation**: COM interface requires native calls
3. **Multi-Monitor**: `MonitorFromWindow` not in `x/sys/windows`

### Zig Advantages:
- Clean C ABI for callback functions
- Static linking of `user32.dll`, `ole32.dll`
- Better cross-compilation support

## Testing Plan

### Test Scenarios:
1. **Foreground Switch**: Switch between apps, verify time tracked correctly
2. **Idle Test**: Leave computer idle 10 minutes, verify pause
3. **Fullscreen Game**: Play game, verify fullscreen detection
4. **Multi-Monitor**: Drag window between monitors, verify tracking
5. **Browser Tabs**: Switch tabs in Chrome, verify URL tracking

### Success Metrics:
- [ ] <1% CPU usage during idle
- [ ] <10ms latency on window focus change
- [ ] 100% accuracy on foreground window detection
- [ ] <50ms URL extraction for browser tabs

## Future Enhancements
- **AI Activity Classification**: Machine learning for "productive" vs "distraction"
- **Heat Maps**: Visualize usage patterns by hour/day
- **Productivity Score**: AI-generated score based on app categories
