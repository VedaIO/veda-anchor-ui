# Advanced Task Scheduling

## Overview
Implement sophisticated time-based rules using Windows Task Scheduler COM API, enabling scenarios like "Allow Minecraft for 30 minutes starting at 3 PM" or "Block YouTube during homework hours (4-6 PM)".

## Current Limitation
**Problem:** Manual, static time limits
- Parent must manually add time
- No scheduled allowances
- No time-of-day restrictions
- No recurring schedules (weekday vs weekend)

## Use Cases

### 1. Scheduled Allowances
```
Rule: "Allow Fortnite for 1 hour from 7-8 PM on weekends only"
  - Game auto-unlocks at 7:00 PM Saturday/Sunday
  - Timer starts, max 1 hour
  - Auto-locks at 8:00 PM or when time expires
```

### 2. School Schedule Blocks
```
Rule: "Block games during school hours (8 AM - 3 PM, Monday-Friday)"
  - All games blocked at 8:00 AM
  - Auto-unlocked at 3:00 PM
  - Does not apply on weekends/holidays
```

### 3. Homework Time
```
Rule: "Limit browser to educational sites 4-6 PM weekdays"
  - Chrome allowed only for school websites
  - YouTube blocked entirely
  - Google Docs, Khan Academy allowed
```

### 4. Bedtime Enforcement
```
Rule: "Block ALL apps after 10 PM"
  - Except: Kindle app (for reading)
  - Auto-shutdown PC at 10:30 PM if still active
```

## Technical Approach

### 1. Windows Task Scheduler COM API
**WinAPI/COM Required:**
- `ITaskService` - Connect to Task Scheduler
- `ITaskFolder` - Manage task folders
- `ITaskDefinition` - Define scheduled tasks
- `ITrigger` - Set time-based triggers

**Architecture:**
```cpp
#include <taskschd.h>

HRESULT CreateScheduledRule(const wchar_t* ruleName, 
                             const wchar_t* executable,
                             SYSTEMTIME startTime, 
                             int durationMinutes) {
    ITaskService* pService = NULL;
    CoCreateInstance(CLSID_TaskScheduler, NULL, CLSCTX_INPROC_SERVER,
                      IID_ITaskService, (void**)&pService);
    
    pService->Connect(_variant_t(), _variant_t(), _variant_t(), _variant_t());
    
    ITaskFolder* pRootFolder = NULL;
    pService->GetFolder(_bstr_t(L"\\ProcGuard"), &pRootFolder);
    
    ITaskDefinition* pTask = NULL;
    pService->NewTask(0, &pTask);
    
    // Set trigger (daily at 7 PM)
    ITriggerCollection* pTriggerCollection = NULL;
    pTask->get_Triggers(&pTriggerCollection);
    
    ITrigger* pTrigger = NULL;
    pTriggerCollection->Create(TASK_TRIGGER_DAILY, &pTrigger);
    
    IDailyTrigger* pDailyTrigger = NULL;
    pTrigger->QueryInterface(IID_IDailyTrigger, (void**)&pDailyTrigger);
    pDailyTrigger->put_StartBoundary(_bstr_t(L"2024-01-01T19:00:00"));
    pDailyTrigger->put_DaysInterval(1);  // Every day
    
    // Set action (run ProcGuard command)
    IActionCollection* pActionCollection = NULL;
    pTask->get_Actions(&pActionCollection);
    
    IAction* pAction = NULL;
    pActionCollection->Create(TASK_ACTION_EXEC, &pAction);
    
    IExecAction* pExecAction = NULL;
    pAction->QueryInterface(IID_IExecAction, (void**)&pExecAction);
    pExecAction->put_Path(_bstr_t(L"C:\\ProcGuard\\procguard-cli.exe"));
    pExecAction->put_Arguments(_bstr_t(L"--unlock Fortnite --duration 60"));
    
    // Register task
    IRegisteredTask* pRegisteredTask = NULL;
    pRootFolder->RegisterTaskDefinition(
        _bstr_t(ruleName),
        pTask,
        TASK_CREATE_OR_UPDATE,
        _variant_t(),
        _variant_t(),
        TASK_LOGON_INTERACTIVE_TOKEN,
        _variant_t(L""),
        &pRegisteredTask
    );
    
    // Cleanup...
    return S_OK;
}
```

### 2. CLI Companion Tool
**For Task Scheduler to execute**

**Go Implementation:**
```go
// procguard-cli.exe
func main() {
    flag.Parse()
    
    switch flag.Arg(0) {
    case "--unlock":
        app := flag.Arg(1)
        duration := flag.Int("duration", 60, "minutes")
        UnlockApp(app, *duration)
    case "--lock":
        app := flag.Arg(1)
        LockApp(app)
    case "--shutdown":
        ShutdownPC()
    }
}

func UnlockApp(appName string, minutes int) {
    // Connect to ProcGuard service via named pipe
    conn, _ := net.Dial("tcp", "localhost:8080")
    json.NewEncoder(conn).Encode(map[string]interface{}{
        "action": "unlock",
        "app": appName,
        "duration_minutes": minutes,
    })
}
```

### 3. Schedule Database Schema

```sql
CREATE TABLE scheduled_rules (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    name TEXT,
    executable_path TEXT,
    action TEXT,  -- 'unlock', 'lock', 'shutdown'
    trigger_type TEXT,  -- 'daily', 'weekly', 'once'
    trigger_time TEXT,  -- ISO 8601
    duration_minutes INTEGER,
    days_of_week TEXT,  -- JSON array [1,2,3,4,5] for Mon-Fri
    is_active BOOLEAN DEFAULT 1,
    task_scheduler_id TEXT,  -- Windows Task GUID
    created_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### 4. Rule Examples in Database

```sql
-- Allow Minecraft 1 hour at 3 PM on weekends
INSERT INTO scheduled_rules VALUES (
    1, 1, 'Weekend Minecraft',
    'C:\Games\Minecraft.exe',
    'unlock', 'weekly', '15:00:00',
    60, '[6,7]', 1, 'task-guid-123', NOW()
);

-- Block YouTube during homework (4-6 PM weekdays)
INSERT INTO scheduled_rules VALUES (
    2, 1, 'Homework Time YouTube Block',
    'C:\Program Files\Google\Chrome\chrome.exe',
    'lock', 'weekly', '16:00:00',
    120, '[1,2,3,4,5]', 1, 'task-guid-456', NOW()
);
```

## Implementation Phases

### Phase 1: CLI Tool (Pure Go)
**Timeline:** 2-3 days
- [ ] Create `procguard-cli.exe`
- [ ] Implement `--unlock`, `--lock` commands
- [ ] IPC with main service
- [ ] Test manual CLI usage

### Phase 2: Task Scheduler Integration (CGO + COM)
**Timeline:** 5-7 days
- [ ] Implement `ITaskService` COM wrapper
- [ ] Create scheduled tasks from rules
- [ ] Update/delete tasks on rule changes
- [ ] Test task execution

### Phase 3: UI for Rule Management (Svelte)
**Timeline:** 4-5 days
- [ ] "Scheduled Rules" page
- [ ] Date/time picker
- [ ] Day-of-week selector
- [ ] Duration slider
- [ ] Test rule creation flow

### Phase 4: Advanced Triggers (CGO + COM)
**Timeline:** 3-4 days
- [ ] One-time schedules
- [ ] Holiday exceptions (skip on Christmas)
- [ ] Conditional triggers (if idle > 10 min)

## UI Design

### Rule Creation Wizard
```svelte
<div class="rule-wizard">
  <h2>Create Scheduled Rule</h2>
  
  <label>App to Control</label>
  <select bind:value={selectedApp}>
    <option>Fortnite</option>
    <option>Minecraft</option>
    <option>YouTube (Chrome)</option>
  </select>
  
  <label>Action</label>
  <div class="radio-group">
    <input type="radio" value="unlock" bind:group={action} /> Allow
    <input type="radio" value="lock" bind:group={action} /> Block
  </div>
  
  <label>Schedule</label>
  <select bind:value={trigger}>
    <option value="daily">Every Day</option>
    <option value="weekdays">Weekdays Only</option>
    <option value="weekends">Weekends Only</option>
    <option value="custom">Custom Days</option>
  </select>
  
  {#if trigger === 'custom'}
    <div class="day-selector">
      <label><input type="checkbox" bind:checked={days[0]} /> Mon</label>
      <label><input type="checkbox" bind:checked={days[1]} /> Tue</label>
      <!-- ... -->
    </div>
  {/if}
  
  <label>Time</label>
  <input type="time" bind:value={startTime} />
  
  <label>Duration</label>
  <input type="range" min="15" max="180" step="15" bind:value={duration} />
  <span>{duration} minutes</span>
  
  <button on:click={createRule}>Create Rule</button>
</div>
```

### Rule List
```svelte
<div class="rules-list">
  {#each rules as rule}
    <div class="rule-card">
      <div class="rule-header">
        <h3>{rule.name}</h3>
        <toggle bind:checked={rule.is_active} />
      </div>
      <p class="rule-details">
        {rule.action === 'unlock' ? 'Allow' : 'Block'} {rule.app_name}
        at {rule.trigger_time} for {rule.duration_minutes} min
      </p>
      <p class="rule-schedule">
        {formatDays(rule.days_of_week)}
      </p>
      <button on:click={() => editRule(rule.id)}>Edit</button>
      <button on:click={() => deleteRule(rule.id)}>Delete</button>
    </div>
  {/each}
</div>
```

## CGO + Zig Benefits

### Why CGO is Needed:
1. **COM Interface:** `ITaskService` requires C++ COM
2. **Complex Structures:** `ITaskDefinition` has 20+ properties
3. **BSTR Handling:** COM strings require `SysFreeString` management

### Zig Advantages:
- Static linking of `taskschd.lib`
- Cleaner COM interface generation
- Cross-compile task management tools

## Smart Features

### 1. Conflict Detection
```
Warning: Rule "Weekend Gaming" overlaps with "Bedtime Block"
  - Weekend Gaming unlocks Fortnite at 9 PM
  - Bedtime Block locks all apps at 10 PM
Suggestion: Adjust timing or create exception
```

### 2. Auto-Extension Requests
```
Child uses app for scheduled 30 minutes
Toast to parent: "Child used all 30 minutes of Minecraft. Allow 30 more?"
  [Yes] -> Extend timer
  [No] -> App blocked
```

### 3. Dynamic Scheduling
```
Rule: "Allow games AFTER homework is done"
  - Detect: Browser usage on school websites
  - If >30 min on homework sites -> Unlock games early
  - Else: Wait until scheduled time
```

## Testing Plan

### Test Cases:
1. **Daily Rule:** Create daily unlock, verify triggers every day
2. **Weekend Rule:** Create weekend-only, verify skips weekdays
3. **Duration Test:** 15-minute rule, verify app locks after time
4. **Conflict Test:** Overlapping rules, verify warning shown
5. **Task Failure:** Disable Task Scheduler service, verify fallback

### Success Metrics:
- [ ] Scheduled unlocks trigger within 1 minute of schedule
- [ ] Duration limits enforced accurately
- [ ] UI clearly shows next scheduled event
- [ ] No zombie tasks in Task Scheduler

## Privacy & Transparency
- All scheduled rules visible to child (read-only)
- Clear countdown timers during scheduled sessions
- Notification before auto-lock occurs

## Future Enhancements
- **AI Scheduling:** Learn child's patterns, suggest optimal schedules
- **Location-Based:** Unlock only when at home (via GPS/WiFi)
- **Conditional Rules:** "If homework done by 5 PM, unlock games at 6 PM"
