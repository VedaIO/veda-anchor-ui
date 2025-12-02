# Anti-Cheat Defense System

## Overview
Detect and report attempts to circumvent ProcGuard by renaming executables, modifying file hashes, or using symbolic links to bypass blocking rules.

## Current Vulnerability
- Users can rename `chrome.exe` to `notchrome.exe` and bypass block
- Copy executables to different directories
- Use symbolic links/junctions to obscure paths
- Modify PE headers to change process name

## Goals
1. **Hash-Based Tracking**: Identify processes by content, not name
2. **Origin Tracking**: Detect file copies and report source
3. **Symbolic Link Detection**: Resolve true executable paths
4. **Tamper Reporting**: Log all circumvention attempts with evidence

## Technical Approach

### 1. Executable Hash Database
**Current (Pure Go - ✅ Already Possible):**
```go
// Calculate SHA256 of executable
func getExecutableHash(path string) (string, error) {
    f, _ := os.Open(path)
    defer f.Close()
    h := sha256.New()
    io.Copy(h, f)
    return hex.EncodeToString(h.Sum(nil)), nil
}
```

**Enhancement:**
- Pre-compute hashes of all blocked executables on first block
- Store in `blocked_hashes` table
- Check new processes against hash database

### 2. PE Authenticode Signature Verification
**WinAPI Required:**
- `WinVerifyTrust` - Verify digital signatures
- `CryptCATAdminCalcHashFromFileHandle` - Get catalog hash

**Use Case:**
```
Blocked: chrome.exe (Google LLC signature)
Detected: notchrome.exe (Google LLC signature) <- SAME BINARY!
Action: Block + Report "Renamed chrome.exe detected"
```

**CGO Implementation:**
```c
#include <wintrust.h>
#include <softpub.h>

BOOL VerifyEmbeddedSignature(LPCWSTR pwszSourceFile) {
    WINTRUST_FILE_INFO FileData = {0};
    FileData.cbStruct = sizeof(WINTRUST_FILE_INFO);
    FileData.pcwszFilePath = pwszSourceFile;
    
    WINTRUST_DATA WinTrustData = {0};
    WinTrustData.cbStruct = sizeof(WinTrustData);
    WinTrustData.dwUIChoice = WTD_UI_NONE;
    WinTrustData.dwUnionChoice = WTD_CHOICE_FILE;
    WinTrustData.pFile = &FileData;
    
    LONG lStatus = WinVerifyTrust(NULL, &PolicyGUID, &WinTrustData);
    return lStatus == ERROR_SUCCESS;
}
```

### 3. Symbolic Link Resolution
**WinAPI Required:**
- `GetFinalPathNameByHandleW` - Resolve symlinks/junctions
- `DeviceIoControl` with `FSCTL_GET_REPARSE_POINT`

**Scenario:**
```
User creates: C:\NotGames\roblox.lnk -> C:\Program Files\Roblox\RobloxPlayerBeta.exe
ProcGuard detects: Process path is symlink, resolves to real path, blocks
```

**CGO Implementation:**
```c
HANDLE hFile = CreateFileW(path, 0, FILE_SHARE_READ, NULL, 
                           OPEN_EXISTING, FILE_FLAG_BACKUP_SEMANTICS, NULL);
WCHAR finalPath[MAX_PATH];
GetFinalPathNameByHandleW(hFile, finalPath, MAX_PATH, FILE_NAME_NORMALIZED);
CloseHandle(hFile);
```

### 4. File Origin Tracking
**WinAPI Required:**
- `GetFileInformationByHandleEx` with `FileIdInfo` - Get file ID
- `FILE_ID_INFO.VolumeSerialNumber` - Track across renames

**Detection Logic:**
```
1. On first block: Record (Hash, VolumeID, FileID, OriginalPath)
2. On process start: Check if (Hash, VolumeID, FileID) matches
3. If match but path differs: TAMPER DETECTED
```

### 5. PE Import Table Analysis
**WinAPI Required:**
- `ImageNtHeader` - Parse PE headers
- `ImageDirectoryEntryToData` - Extract import table

**Use Case:**
Detect if executable imports suspicious DLLs:
- `xinput1_3.dll` (game controller)
- `d3d11.dll` (DirectX - likely a game)
- `steam_api64.dll` (Steam game)

### 6. Process Ancestry Tracking
**WinAPI Required:**
- `NtQueryInformationProcess` with `ProcessBasicInformation`
- `PROCESS_BASIC_INFORMATION.InheritedFromUniqueProcessId`

**Detection:**
```
If new suspicious process spawned from explorer.exe:
  -> User likely double-clicked a renamed exe
  -> Report: "Suspicious executable launched from Desktop"
```

## Implementation Phases

### Phase 1: Hash-Based Blocking (Pure Go)
**Timeline:** 3-4 days
- [x] Calculate SHA256 on first block
- [ ] Store in database
- [ ] Check hash on every process start
- [ ] Report hash mismatch events

### Phase 2: Symlink Resolution (Pure Go)
**Timeline:** 2-3 days
- [ ] Use `os.Readlink` for basic symlink resolution
- [ ] Resolve junctions via `syscall` package
- [ ] Normalize all paths before comparison

### Phase 3: Signature Verification (CGO Required)
**Timeline:** 5-6 days
- [ ] Implement `WinVerifyTrust` wrapper
- [ ] Extract publisher info from certificate
- [ ] Match blocked apps by publisher + product name
- [ ] Report: "Adobe Photoshop detected (renamed from photoshop.exe)"

### Phase 4: Advanced PE Analysis (CGO Required)
**Timeline:** 4-5 days
- [ ] Parse import tables
- [ ] Detect game-related DLLs
- [ ] Flag suspicious binaries auto-detected
- [ ] Machine learning model for classification (optional)

### Phase 5: Real-Time Monitoring (CGO + ETW)
**Timeline:** 7-10 days
- [ ] Subscribe to ETW `FileCreate` events
- [ ] Detect when blocked exe is copied
- [ ] Immediate alert: "Roblox copied to C:\NotGames\"

## Reporting System

### Tamper Event Structure
```go
type TamperEvent struct {
    Timestamp      time.Time
    OriginalPath   string
    DetectedPath   string
    Hash           string
    TamperType     string // "rename", "copy", "symlink"
    Publisher      string // From authenticode
    Evidence       []byte // Screenshot/file metadata
}
```

### Report Delivery
1. **In-App Notification**: Toast notification for parent
2. **Email Alert**: Send to parent's email (if configured)
3. **Database Log**: Permanent record with evidence
4. **Screenshot Capture**: Optional - capture screen at detection time

## CGO + Zig Benefits

### Why CGO is Needed:
1. **WinVerifyTrust**: No pure Go equivalent, must call `wintrust.dll`
2. **PE Parsing**: `imagehlp.dll` is more reliable than manual parsing
3. **File IDs**: `GetFileInformationByHandleEx` requires native handle

### Zig Advantages:
- Clean FFI for Windows SDK structs (`WINTRUST_DATA` is complex)
- Static linking to avoid DLL dependency issues
- Better error handling for syscall failures

## Testing Plan

### Test Cases:
1. **Rename Test**: Block `chrome.exe`, rename to `notchrome.exe`, verify detection
2. **Copy Test**: Block game, copy to `C:\Temp\game.exe`, verify detection
3. **Symlink Test**: Create junction to blocked exe, verify resolution
4. **Hash Collision Test**: Ensure different apps with same hash handled correctly
5. **False Positive Test**: Verify legitimate renamed apps still work

### Success Metrics:
- [ ] 100% detection rate for renamed executables (via hash)
- [ ] < 0.1% false positive rate
- [ ] < 100ms detection latency
- [ ] All tamper events logged with evidence

## Privacy Considerations
- Hash database stays local (never uploaded)
- Parent can review tamper logs before taking action
- Option to disable advanced tracking for privacy-conscious users

## Future Enhancements
- **Behavioral Analysis**: Detect if renamed calc.exe acts like a game (GPU usage spike)
- **Cloud Hash Database**: Crowdsourced hash database of common apps
- **YARA Rules**: Signature-based detection for game engines
