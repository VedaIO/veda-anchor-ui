# Network Filtering (Windows Filtering Platform)

## Overview
**Priority: Nice-to-Have (Second-Class Feature)**

Implement per-application network access control using Windows Filtering Platform (WFP) to selectively block internet access for specific apps without affecting others.

## Use Cases

### 1. Offline Mode for Distracting Apps
```
Allow Photoshop to run, but block its internet access:
  - Prevents update notifications
  - Blocks Adobe Creative Cloud sync
  - Keeps app functional offline
```

### 2. Study Mode
```
Block ALL internet except:
  - Zoom (for online classes)
  - Google Docs
  - School learning portals
```

### 3. Data Usage Tracking
```
Track network usage per app:
  - Fortnite: 2.3 GB today
  - YouTube: 5.1 GB today
  - Warn parent if excessive
```

## Technical Approach

### 1. Windows Filtering Platform (WFP) API
**WinAPI Required:**
- `FwpmEngineOpen0` - Connect to firewall engine
- `FwpmFilterAdd0` - Add firewall filter
- `FwpmCalloutAdd0` - Register custom callout driver

**Architecture:**
```c
#include <fwpmu.h>
#include <fwpmtypes.h>

typedef struct {
    UINT64 filterId;
    WCHAR appPath[MAX_PATH];
    BOOL isBlocked;
} AppNetworkFilter;

DWORD BlockAppNetwork(const WCHAR* appPath) {
    HANDLE engineHandle;
    FWPM_SESSION0 session = {0};
    
    // Open WFP engine
    FwpmEngineOpen0(NULL, RPC_C_AUTHN_WINNT, NULL, &session, &engineHandle);
    
    // Create filter for outbound IPv4
    FWPM_FILTER0 filter = {0};
    filter.layerKey = FWPM_LAYER_ALE_AUTH_CONNECT_V4;
    filter.action.type = FWP_ACTION_BLOCK;
    filter.filterCondition = /* Match app path */;
    filter.weight.type = FWP_UINT8;
    filter.weight.uint8 = 15;  // High priority
    
    UINT64 filterId;
    DWORD result = FwpmFilterAdd0(engineHandle, &filter, NULL, &filterId);
    
    FwpmEngineClose0(engineHandle);
    return result;
}
```

### 2. Callout Driver (Advanced - Kernel Mode)
**For packet inspection & data usage tracking**

**WinAPI/WDK Required:**
- `FwpsCalloutRegister0` - Register kernel callout
- `WdfDriverCreate` - Create WDF driver

**Use Case:**
- Intercept packets at kernel level
- Count bytes sent/received per app
- Deep packet inspection (detect VPN bypass attempts)

**Note:** Requires signed kernel driver ($$$ EV certificate)

### 3. User-Mode Filtering (Simpler Alternative)

**WinAPI Required:**
- `FwpmFilterAdd0` with `FWPM_LAYER_ALE_AUTH_CONNECT_V4`
- No kernel driver needed

**Limitations:**
- Cannot inspect packet contents
- Can only block/allow by IP/port/app
- Cannot track data usage accurately (requires callout)

## Implementation Phases

### Phase 1: Basic App Blocking (CGO Required)
**Timeline:** 5-7 days
- [ ] Implement `FwpmEngineOpen0` wrapper
- [ ] Add filters for blocked apps
- [ ] Test blocking Chrome while allowing Firefox
- [ ] **Complexity:** Medium (complex WFP structures)

### Phase 2: Port/IP Whitelist (CGO Required)
**Timeline:** 3-4 days
- [ ] Allow specific IPs for blocked apps (e.g., Zoom servers only)
- [ ] Port-based filtering (block HTTP/HTTPS, allow custom ports)
- [ ] Dynamic IP updates (e.g., for Zoom's changing servers)

### Phase 3: Data Usage Tracking (Kernel Driver - HARD)
**Timeline:** 15-20 days + $300 cert
- [ ] Develop WFP callout driver
- [ ] Count bytes per app
- [ ] Persist stats to database
- [ ] **Requires:** Kernel mode programming, driver signing

### Phase 4: UI Integration (Pure Go)
**Timeline:** 2-3 days
- [ ] "Block Network" toggle in app details
- [ ] Data usage graphs
- [ ] Manual IP/Port whitelist UI

## Database Schema

```sql
CREATE TABLE network_filters (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    executable_path TEXT,
    is_blocked BOOLEAN,
    allowed_ips TEXT,  -- JSON array
    allowed_ports TEXT,  -- JSON array
    created_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE network_usage (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    executable_path TEXT,
    date DATE,
    bytes_sent INTEGER,
    bytes_received INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## CGO + Zig Benefits

### Why CGO is Needed:
1. **WFP Structures:** `FWPM_FILTER0` has 20+ fields with complex unions
2. **Kernel Driver:** Callout driver requires C/C++ with WDK
3. **Privilege Escalation:** Modifying firewall requires `SeSecurityPrivilege`

### Zig Advantages:
- Static linking of `fwpuclnt.lib`
- Cleaner FFI for nested WFP structures
- Cross-compile management interface

## Testing Plan

### Test Cases:
1. **Single App Block:** Block Chrome, verify Firefox works
2. **Whitelist Test:** Block Zoom except specific IPs
3. **Port Filter:** Allow app on port 443, block port 80
4. **Multi-User:** User A blocks, User B unaffected
5. **VPN Detection:** Verify filtering works with VPN active

### Success Metrics:
- [ ] 100% block rate for targeted apps
- [ ] <5ms latency added to connections
- [ ] No system-wide network disruption

## Challenges & Risks

### 1. VPN Bypass
**Problem:** App uses VPN to tunnel past WFP
**Mitigation:** Detect VPN adapters, warn parent

### 2. False Positives
**Problem:** Blocking Electron app blocks all Chromium-based apps
**Mitigation:** Use process signature verification

### 3. Certificate Cost
**Problem:** Kernel driver requires EV certificate (~$300/year)
**Decision:** Phase 3 is optional, Phase 1-2 sufficient for most users

## Future Enhancements
- **Content Filtering:** Block specific websites (not just apps)
- **AI-Based Detection:** Detect gaming traffic patterns
- **Cloud Sync:** Share allowed IP lists across ProcGuard users
