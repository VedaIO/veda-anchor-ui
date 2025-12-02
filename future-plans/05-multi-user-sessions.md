# Multi-User Session Handling

## Overview
Support multiple Windows user accounts on the same machine, tracking time limits and blocks independently per user. Handle scenarios like work computers (admin + standard user), parent computers (parent + child accounts), and school computers (teacher + student accounts).

## Current Limitation
**Problem:** ProcGuard runs as single-instance service
- Time limits are global (not per-user)
- If child logs in, then parent logs in (Fast User Switching), timers get confused
- No distinction between admin and standard users

## Use Cases

### 1. Home Scenario
```
PC Users:
  - Parent (Administrator)
  - Child (Standard User)

Expected Behavior:
  - Parent: No restrictions
  - Child: 2h/day max screen time for games
```

### 2. Work Scenario
```
PC Users:
  - Manager (Administrator)
  - Employee (Standard User)

Expected Behavior:
  - Manager: Full control of ProcGuard
  - Employee: Blocked from personal apps during work hours
```

### 3. School Scenario
```
PC Users:
  - Teacher (Administrator)
  - Student 1 (Standard User)
  - Student 2 (Standard User)

Expected Behavior:
  - Teacher: Configure blocks, view all student activity
  - Students: Independent time limits, cannot interfere with each other
```

## Technical Approach

### 1. Session Detection
**WinAPI Required:**
- `WTSEnumerateSessionsW` - List all active sessions
- `WTSQuerySessionInformationW` - Get session details (username, state)
- `ProcessIdToSessionId` - Map process to session

**Architecture:**
```c
#include <wtsapi32.h>

typedef struct {
    DWORD sessionId;
    WCHAR username[256];
    WTS_CONNECTSTATE_CLASS state;
    BOOL isAdmin;
} SessionInfo;

SessionInfo* EnumerateSessions(DWORD* count) {
    WTS_SESSION_INFOW* pSessionInfo;
    DWORD sessionCount;
    
    WTSEnumerateSessionsW(WTS_CURRENT_SERVER_HANDLE, 0, 1, 
                          &pSessionInfo, &sessionCount);
    
    SessionInfo* sessions = malloc(sessionCount * sizeof(SessionInfo));
    
    for (DWORD i = 0; i < sessionCount; i++) {
        sessions[i].sessionId = pSessionInfo[i].SessionId;
        
        LPWSTR username;
        DWORD bytes;
        WTSQuerySessionInformationW(WTS_CURRENT_SERVER_HANDLE, 
                                    pSessionInfo[i].SessionId,
                                    WTSUserName, &username, &bytes);
        wcscpy(sessions[i].username, username);
        WTSFreeMemory(username);
        
        sessions[i].state = pSessionInfo[i].State;
        sessions[i].isAdmin = IsUserAdmin(sessions[i].sessionId);
    }
    
    *count = sessionCount;
    WTSFreeMemory(pSessionInfo);
    return sessions;
}
```

### 2. User Privilege Detection
**WinAPI Required:**
- `OpenProcessToken` - Get process token
- `GetTokenInformation` with `TokenElevation` - Check if elevated
- `CheckTokenMembership` for `Administrators` SID

**Implementation:**
```c
BOOL IsUserAdmin(DWORD sessionId) {
    HANDLE hToken;
    WTS_SESSIONS_HANDLE hServer = WTS_CURRENT_SERVER_HANDLE;
    
    if (!WTSQueryUserToken(sessionId, &hToken)) return FALSE;
    
    SID_IDENTIFIER_AUTHORITY NtAuthority = SECURITY_NT_AUTHORITY;
    PSID AdministratorsGroup;
    AllocateAndInitializeSid(&NtAuthority, 2,
        SECURITY_BUILTIN_DOMAIN_RID,
        DOMAIN_ALIAS_RID_ADMINS,
        0, 0, 0, 0, 0, 0, &AdministratorsGroup);
    
    BOOL isAdmin = FALSE;
    CheckTokenMembership(hToken, AdministratorsGroup, &isAdmin);
    
    FreeSid(AdministratorsGroup);
    CloseHandle(hToken);
    return isAdmin;
}
```

### 3. Per-User Database Schema

**Users Table:**
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    sid TEXT UNIQUE,  -- Windows Security Identifier
    is_admin BOOLEAN DEFAULT 0,
    is_restricted BOOLEAN DEFAULT 1,
    daily_limit_seconds INTEGER,  -- NULL = no limit
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**User Sessions Table:**
```sql
CREATE TABLE user_sessions (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    session_id INTEGER,
    login_time TIMESTAMP,
    logout_time TIMESTAMP,
    is_active BOOLEAN DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

**Per-User App History:**
```sql
CREATE TABLE app_history (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,  -- NEW: Track per user
    session_id INTEGER,  -- NEW: Track per session
    executable_path TEXT,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    duration_seconds INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

**Per-User Blocks:**
```sql
CREATE TABLE user_blocks (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    executable_path TEXT,
    reason TEXT,
    created_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### 4. Session Monitoring Service

**Architecture:**
```
ProcGuard Service (SYSTEM account)
  ├─> Session Monitor Thread
  │     ├─> Polls WTSEnumerateSessions every 5 seconds
  │     ├─> Detects new logins
  │     ├─> Detects logouts/disconnects
  │     ├─> Detects Fast User Switching
  │     └─> Updates user_sessions table
  │
  ├─> Per-Session Process Monitor
  │     ├─> Thread for Session 1 (Parent)
  │     ├─> Thread for Session 2 (Child)
  │     └─> Each thread tracks processes in its session
  │
  └─> UI Instance per Session
        ├─> Launches Wails UI in user's session
        └─> Shows user-specific data
```

**CGO Implementation:**
```c
void MonitorSessions() {
    while (1) {
        DWORD count;
        SessionInfo* sessions = EnumerateSessions(&count);
        
        for (DWORD i = 0; i < count; i++) {
            if (sessions[i].state == WTSActive) {
                // Active session detected
                NotifyGoSessionActive(sessions[i].sessionId, 
                                      sessions[i].username);
            } else if (sessions[i].state == WTSDisconnected) {
                // User switched or locked
                NotifyGoSessionDisconnected(sessions[i].sessionId);
            }
        }
        
        free(sessions);
        Sleep(5000);  // Poll every 5 seconds
    }
}
```

### 5. Process-to-Session Mapping
**WinAPI Required:**
- `ProcessIdToSessionId` - Get session ID for process

**Usage:**
```go
func GetProcessSession(pid uint32) (uint32, error) {
    var sessionID uint32
    err := ProcessIdToSessionId(pid, &sessionID)
    return sessionID, err
}

func TrackProcess(pid uint32) {
    sessionID, _ := GetProcessSession(pid)
    user := GetUserForSession(sessionID)
    
    // Check if user has this app blocked
    if IsBlockedForUser(user.ID, exePath) {
        KillProcess(pid)
        LogBlock(user.ID, exePath)
    }
}
```

### 6. Fast User Switching Handling

**Scenario:**
```
1. Child logs in (Session 1)
2. Child plays Roblox for 30 minutes
3. Parent presses Win+L, switches to parent account (Session 2)
4. Roblox still running in Session 1 (suspended)
5. Parent uses computer for 1 hour
6. Parent switches back to child (Session 1)
7. Roblox resumes

Expected: Child's timer continues from 30 minutes, not 1h30m
```

**Implementation:**
```go
type SessionTracker struct {
    ActiveSession uint32
    UserTimers    map[uint32]*TimeTracker
}

func (st *SessionTracker) OnSessionChange(newSessionID uint32) {
    if st.ActiveSession != 0 {
        // Pause previous session's timers
        st.UserTimers[st.ActiveSession].Pause()
    }
    
    st.ActiveSession = newSessionID
    
    if timer, exists := st.UserTimers[newSessionID]; exists {
        // Resume this session's timers
        timer.Resume()
    }
}
```

### 7. UI Instance Management

**WinAPI Required:**
- `WTSQueryUserToken` - Get user token for session
- `CreateProcessAsUserW` - Launch UI in user's session

**Launch UI per User:**
```c
BOOL LaunchUIForSession(DWORD sessionId) {
    HANDLE hToken;
    if (!WTSQueryUserToken(sessionId, &hToken)) return FALSE;
    
    STARTUPINFOW si = {0};
    PROCESS_INFORMATION pi = {0};
    si.cb = sizeof(si);
    si.lpDesktop = L"winsta0\\default";
    
    WCHAR cmdLine[] = L"C:\\Program Files\\ProcGuard\\wails-app.exe";
    
    // Create process in user's session with user's token
    BOOL result = CreateProcessAsUserW(
        hToken,
        NULL,
        cmdLine,
        NULL, NULL, FALSE,
        CREATE_NEW_CONSOLE | CREATE_UNICODE_ENVIRONMENT,
        NULL, NULL,
        &si, &pi
    );
    
    CloseHandle(hToken);
    CloseHandle(pi.hProcess);
    CloseHandle(pi.hThread);
    return result;
}
```

## Implementation Phases

### Phase 1: Session Detection (CGO Required)
**Timeline:** 3-4 days
- [ ] Implement `WTSEnumerateSessions` wrapper
- [ ] Detect active sessions
- [ ] Map processes to sessions
- [ ] Log session changes

### Phase 2: User Identification (CGO Required)
**Timeline:** 2-3 days
- [ ] Get username per session
- [ ] Detect admin vs standard user
- [ ] Create users table schema
- [ ] Auto-create user records on login

### Phase 3: Per-User Database (Pure Go)
**Timeline:** 3-4 days
- [ ] Add `user_id` to all tracking tables
- [ ] Migrate existing data to "default" user
- [ ] Update queries to filter by user
- [ ] UI shows current user's data only

### Phase 4: Per-Session UI (CGO Required)
**Timeline:** 4-5 days
- [ ] Implement `CreateProcessAsUserW`
- [ ] Launch UI instance per session
- [ ] IPC between service and UI instances
- [ ] Test multi-session scenarios

### Phase 5: Fast User Switching (Pure Go + CGO)
**Timeline:** 3-4 days
- [ ] Detect session state changes
- [ ] Pause/resume timers correctly
- [ ] Test switching scenarios
- [ ] Handle crashed sessions

## Configuration UI

### User Management Screen
```svelte
<div class="user-management">
  <h2>Managed Users</h2>
  <table>
    <tr>
      <th>Username</th>
      <th>Type</th>
      <th>Daily Limit</th>
      <th>Actions</th>
    </tr>
    <tr>
      <td>Parent</td>
      <td>Administrator</td>
      <td>None</td>
      <td><button disabled>Edit</button></td>
    </tr>
    <tr>
      <td>Child</td>
      <td>Standard User</td>
      <td>2 hours</td>
      <td><button>Edit Limits</button></td>
    </tr>
  </table>
</div>
```

### Per-User Block List
```
User: Child
Blocked Apps:
  - Roblox
  - Fortnite
  - Discord

User: Parent
Blocked Apps: (none)
```

## CGO + Zig Benefits

### Why CGO is Needed:
1. **Session Enumeration**: `WTSEnumerateSessions` not in `x/sys/windows`
2. **User Token Handling**: `WTSQueryUserToken` complex token operations
3. **Process Creation**: `CreateProcessAsUserW` requires privilege management

### Zig Advantages:
- Static linking of `wtsapi32.lib`
- Cleaner FFI for session structures
- Cross-compile service from Linux/Mac

## Testing Plan

### Test Scenarios:
1. **Multi-User Login**: Login as Parent, then Child, verify independent timers
2. **Fast User Switch**: Switch between users, verify timers pause/resume
3. **Concurrent Sessions**: Two users logged in, verify both tracked separately
4. **Admin Detection**: Verify parent (admin) has no restrictions
5. **Session Crash**: Kill child session, verify parent unaffected

### Success Metrics:
- [ ] 100% accurate session-to-user mapping
- [ ] <5 second detection latency for new logins
- [ ] Zero timer bleed between users
- [ ] UI launches in correct user's session

## Privacy & Security

### Isolation:
- Users cannot see each other's app history
- Only administrators can view all users
- Separate SQLite databases per user (optional)

### Data Retention:
- Admin can configure: "Delete student data on logout"
- Useful for school computers (privacy compliance)

## Future Enhancements
- **Active Directory Integration**: Sync with domain users
- **Profile Syncing**: Roaming profiles across multiple PCs
- **Family Safety Integration**: Sync with Microsoft Family Safety accounts
