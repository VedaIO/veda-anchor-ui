# Windows Credential Manager Integration

## Overview
**Priority: Nice-to-Have**

Securely store ProcGuard passwords in Windows Credential Manager (Vault) instead of bcrypt-hashed database. Leverage OS-level encryption and credential roaming for multi-device scenarios.

## Current Limitation
**Problem:** Passwords stored in `procguard.db` with bcrypt
- Database can be copied/transferred
- No OS-level protection
- No password roaming
- Extra attack surface (DB file theft)

## Benefits

### 1. OS-Level Security
- Windows encrypts credentials with user's DPAPI key
- Tied to user account (cannot be transferred to another PC)
- Survives database deletion

### 2. Centralized Management
- Parents can reset password via Windows if needed
- Integration with Windows Hello (fingerprint/face unlock)
- Credential roaming for domain accounts

### 3. Compliance
- GDPR-friendly (no plaintext storage)
- Meets enterprise security requirements

## Technical Approach

### 1. Basic Credential Storage
**WinAPI Required:**
- `CredWriteW` - Store credential
- `CredReadW` - Retrieve credential
- `CredDeleteW` - Delete credential

**CGO Implementation:**
```c
#include <wincred.h>

BOOL StoreProcGuardPassword(const WCHAR* username, const WCHAR* password) {
    CREDENTIALW cred = {0};
    
    cred.Type = CRED_TYPE_GENERIC;
    cred.TargetName = L"ProcGuard:MasterPassword";
    cred.CredentialBlobSize = (wcslen(password) + 1) * sizeof(WCHAR);
    cred.CredentialBlob = (LPBYTE)password;
    cred.Persist = CRED_PERSIST_LOCAL_MACHINE;  // System-wide
    cred.UserName = (LPWSTR)username;
    cred.Comment = L"ProcGuard Master Password";
    
    return CredWriteW(&cred, 0);
}

BOOL VerifyProcGuardPassword(const WCHAR* username, const WCHAR* password) {
    PCREDENTIALW pCred;
    
    if (!CredReadW(L"ProcGuard:MasterPassword", CRED_TYPE_GENERIC, 0, &pCred)) {
        return FALSE;  // No password set
    }
    
    BOOL match = (wcscmp((WCHAR*)pCred->CredentialBlob, password) == 0);
    CredFree(pCred);
    return match;
}
```

### 2. Windows Hello Integration
**WinAPI Required:**
- `UserConsentVerifierCheckAvailabilityAsync` - Check if Hello available
- `UserConsentVerifierRequestVerificationAsync` - Prompt for fingerprint/face

**Use Case:**
```
Parent sets up password -> Also enrolls fingerprint
Later: Open ProcGuard -> Prompt: "Scan finger to unlock"
```

**Implementation (WinRT/C++):**
```cpp
#include <winrt/Windows.Security.Credentials.UI.h>

using namespace winrt::Windows::Security::Credentials::UI;

async Task<bool> UnlockWithWindowsHello() {
    auto availability = co_await UserConsentVerifier::CheckAvailabilityAsync();
    
    if (availability == UserConsentVerifierAvailability::Available) {
        auto result = co_await UserConsentVerifier::RequestVerificationAsync(
            L"Unlock ProcGuard"
        );
        
        return result.verification == UserConsentVerificationResult::Verified;
    }
    
    return false;
}
```

### 3. DPAPI for Sensitive Data
**WinAPI Required:**
- `CryptProtectData` - Encrypt data with user key
- `CryptUnprotectData` - Decrypt data

**Use Case:** Encrypt database file itself
```c
#include <dpapi.h>

BOOL EncryptDatabaseFile(BYTE* plainData, DWORD plainSize, 
                         BYTE** encryptedData, DWORD* encryptedSize) {
    DATA_BLOB input, output;
    input.pbData = plainData;
    input.cbData = plainSize;
    
    if (CryptProtectData(&input, L"ProcGuard Database", NULL, NULL, NULL, 
                         CRYPTPROTECT_UI_FORBIDDEN, &output)) {
        *encryptedData = output.pbData;
        *encryptedSize = output.cbData;
        return TRUE;
    }
    return FALSE;
}
```

## Implementation Phases

### Phase 1: Replace bcrypt with Credential Manager (CGO)
**Timeline:** 2-3 days
- [ ] Implement `CredWrite`/`CredRead` wrappers
- [ ] Migrate existing passwords to Vault
- [ ] Remove password column from database
- [ ] Test password verification

### Phase 2: Windows Hello Support (CGO + WinRT)
**Timeline:** 4-5 days
- [ ] Check for Windows Hello availability
- [ ] Enroll biometrics on password creation
- [ ] Add "Unlock with Hello" button in UI
- [ ] Fallback to password if Hello fails

### Phase 3: Database Encryption (CGO)
**Timeline:** 3-4 days
- [ ] Encrypt `procguard.db` with `CryptProtectData`
- [ ] Decrypt on startup
- [ ] Re-encrypt on modifications
- [ ] Test file tampering detection

## Database Schema Changes

**Before:**
```sql
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT
);
-- Password stored as: bcrypt_hash
```

**After:**
```sql
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT
);
-- No password storage needed!
-- Windows Credential Manager handles it
```

## UI Changes

### Login Screen (Enhanced)
```svelte
<div class="login">
  <h1>Unlock ProcGuard</h1>
  
  {#if windowsHelloAvailable}
    <button on:click={unlockWithHello}>
      <FingerprintIcon />
      Use Windows Hello
    </button>
    <div class="separator">or</div>
  {/if}
  
  <input type="password" bind:value={password} placeholder="Password" />
  <button on:click={unlockWithPassword}>Unlock</button>
</div>
```

## CGO + Zig Benefits

### Why CGO is Needed:
1. **Credential Manager:** `wincred.h` not in pure Go
2. **DPAPI:** `CryptProtectData` requires native calls
3. **WinRT:** Windows Hello requires C++/WinRT (complex FFI)

### Zig Advantages:
- Static linking of `advapi32.lib`, `crypt32.lib`
- Cleaner struct handling for `CREDENTIALW`
- Cross-compile credential management tools

## Security Considerations

### 1. Credential Roaming
**Benefit:** Password syncs across parent's devices (if domain)
**Risk:** If parent's account compromised, all devices exposed
**Mitigation:** Optional local-only persistence

### 2. Admin Reset
**Benefit:** Parent can reset via Windows if forgotten
**Risk:** Admin user can reset child's ProcGuard password
**Mitigation:** Require parent email verification for reset

### 3. Database Theft
**Benefit:** Encrypted database useless without user account
**Risk:** If user account compromised, database readable
**Mitigation:** Two-factor authentication for sensitive operations

## Testing Plan

### Test Cases:
1. **First-Time Setup:** Set password, verify stored in Vault
2. **Login Test:** Verify password works
3. **Wrong Password:** Verify rejection
4. **Windows Hello:** Enroll finger, unlock with biometric
5. **DB Encryption:** Copy `procguard.db`, verify unreadable on other PC

### Success Metrics:
- [ ] Password stored only in Credential Manager
- [ ] Windows Hello works on supported devices
- [ ] Database encrypted with DPAPI
- [ ] No password leaks in logs/memory dumps

## Privacy Considerations
- Password never touches database (even hashed)
- Credential Manager logs accessible only to SYSTEM
- Biometric data stays in Trusted Platform Module (TPM)

## Future Enhancements
- **Azure AD Sync:** Enterprise password policy enforcement
- **Smart Card
 Support:** Login with YubiKey
- **Multi-Factor Auth:** SMS/Email codes for ultra-secure mode
