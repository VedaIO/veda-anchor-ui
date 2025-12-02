# Self-Protection System

## Overview
Implement comprehensive self-protection mechanisms to ensure ProcGuard cannot be terminated except through the authorized "Dừng ProcGuard" button in the UI.

## Current Vulnerability
- Users can terminate ProcGuard via Task Manager
- Process can be killed with `taskkill /F /IM procguard.exe`
- Registry/service can be modified to disable startup

## Goals
1. **Process Protection**: Prevent unauthorized termination
2. **Service Protection**: Make ProcGuard service tampering-proof
3. **Registry Protection**: Lock down configuration keys
4. **File Protection**: Prevent deletion/modification of executable

## Technical Approach

### 1. Protected Process Light (PPL)
**WinAPI Required:**
- `RtlSetProcessIsCritical` - Mark process as critical (BSOD on termination)
- `NtSetInformationProcess` with `ProcessProtectionInformation`
- Sign executable with EV certificate for PPL-AntiMalware level

**Implementation:**
```c
// Via CGO
typedef enum _PROCESS_PROTECTION_LEVEL {
    PROTECTION_LEVEL_NONE = 0,
    PROTECTION_LEVEL_SAME = 1,
    PROTECTION_LEVEL_PPL_APP = 2
} PROCESS_PROTECTION_LEVEL;

NTSTATUS status = NtSetInformationProcess(
    GetCurrentProcess(),
    ProcessProtectionInformation,
    &protectionInfo,
    sizeof(protectionInfo)
);
```

### 2. Process ACL Hardening
**WinAPI Required:**
- `SetSecurityInfo` - Modify process DACL
- `AdjustTokenPrivileges` - Grant `SeTcbPrivilege`

**Strategy:**
- Deny `PROCESS_TERMINATE` access to all users except SYSTEM
- Allow only the ProcGuard UI to signal graceful shutdown via named event

### 3. Service Watchdog
**WinAPI Required:**
- `ChangeServiceConfig2` - Set failure actions
- `SERVICE_FAILURE_ACTIONS_FLAG` - Restart on crash
- `OpenServiceW` + `QueryServiceStatus` - Monitor health

**Design:**
- Dual-process architecture: UI + Service
- Service monitors UI, UI monitors service
- Mutual resurrection if either crashes

### 4. Registry Protection
**WinAPI Required:**
- `RegSetKeySecurity` - Lock registry keys
- `RegNotifyChangeKeyValue` - Detect tampering attempts

**Protected Keys:**
```
HKLM\SOFTWARE\ProcGuard\*
HKCU\Software\Microsoft\Windows\CurrentVersion\Run\ProcGuard
```

### 5. File System Protection
**WinAPI Required:**
- `SetFileSecurityW` - Deny DELETE/WRITE_DAC
- `FindFirstChangeNotificationW` - Monitor directory changes

**Protected Files:**
- `procguard.exe`
- `procguard-service.exe`
- `procguard.db`

## Implementation Phases

### Phase 1: Basic Process Protection (Pure Go)
- Implement graceful shutdown signal via named event
- Validate shutdown requests with password
- **Effort:** 2-3 days

### Phase 2: Service Hardening (Pure Go)
- Convert to Windows Service with failure recovery
- Add mutual watchdog between UI and Service
- **Effort:** 3-4 days

### Phase 3: ACL Protection (CGO Required)
- Implement process DACL modification
- Add registry key protection
- **Effort:** 4-5 days (requires CGO setup)

### Phase 4: PPL Implementation (CGO + Code Signing)
- Obtain EV certificate (~$300/year)
- Implement PPL via `NtSetInformationProcess`
- **Effort:** 5-7 days + certificate cost

## CGO + Zig Benefits

### Why CGO is Needed:
1. **Undocumented APIs**: `NtSetInformationProcess` not in `x/sys/windows`
2. **Complex Structures**: `PROCESS_PROTECTION_INFORMATION` requires careful packing
3. **Privilege Manipulation**: `SeTcbPrivilege` easier via native calls

### Zig Advantages:
- Cross-compile Windows drivers from Linux (if kernel-level protection needed)
- Easier static linking of Windows SDK
- Better struct packing control for undocumented structures

## Security Considerations

### Anti-Virus False Positives
- PPL + self-resurrection = likely AV flags
- **Mitigation**: Submit to Microsoft SmartScreen, sign with EV cert

### Bootkit/Rootkit Vulnerability
- Advanced users with kernel access can still kill ProcGuard
- **Acceptance**: School/home parental control, not nation-state defense

### Ethical Concerns
- Make uninstall path via built-in menu clear
- Document emergency recovery (Safe Mode boot)

## Testing Plan
1. **Task Manager Kill Test**: Verify denial
2. **taskkill Test**: Verify denial
3. **Registry Modification Test**: Verify blocking
4. **Service Stop Test**: Verify resurrection
5. **Safe Mode Test**: Verify graceful degradation

## Success Metrics
- [ ] Task Manager shows "Access Denied" on terminate
- [ ] `taskkill` returns error code 1
- [ ] Service auto-restarts within 5 seconds of crash
- [ ] Authorized shutdown via UI works reliably
- [ ] Zero false positive AV detections (post-signing)

## Future Enhancements
- Kernel driver for ultimate protection (Phase 5)
- Signed kernel callback for process creation blocking
- Integration with Windows Defender Application Guard
