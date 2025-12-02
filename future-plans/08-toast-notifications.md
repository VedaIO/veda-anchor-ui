# Windows Native Toast Notifications

## Overview
**Priority: Nice-to-Have**

Implement native Windows 10/11 toast notifications (Action Center) instead of in-app web toasts. Provides system-level alerts even when ProcGuard UI is closed, with action buttons for quick responses.

## Current Limitation
**Problem:** In-app toasts only visible when UI open
- Parent misses alerts if UI minimized
- No actionable notifications
- Looks like a web app, not native

## Benefits

### 1. System-Level Alerts
- Notifications show in Action Center
- Persist even if UI closed
- Parent sees alerts instantly

### 2. Actionable Notifications
```
Toast: "Child blocked Roblox (23:47)"
Actions: [View Details] [Add 30 Minutes]
```

### 3. Native Feel
- Matches Windows design language
- Supports images, sounds, buttons
- Works with Focus Assist (Do Not Disturb)

## Technical Approach

### 1. WinRT Toast Notification API
**WinAPI/WinRT Required:**
- `Windows.UI.Notifications.ToastNotificationManager`
- `Windows.UI.Notifications.ToastNotification`
- `Windows.Data.Xml.Dom.XmlDocument`

**Architecture:**
```cpp
#include <winrt/Windows.UI.Notifications.h>
#include <winrt/Windows.Data.Xml.Dom.h>

using namespace winrt;
using namespace Windows::UI::Notifications;
using namespace Windows::Data::Xml::Dom;

void ShowToast(const wchar_t* title, const wchar_t* message) {
    // XML template for toast
    auto toastXml = ToastNotificationManager::GetTemplateContent(
        ToastTemplateType::ToastText02
    );
    
    // Set text
    auto textNodes = toastXml.GetElementsByTagName(L"text");
    textNodes.Item(0).AppendChild(toastXml.CreateTextNode(title));
    textNodes.Item(1).AppendChild(toastXml.CreateTextNode(message));
    
    // Create notification
    auto toast = ToastNotification(toastXml);
    
    // Show toast
    auto notifier = ToastNotificationManager::CreateToastNotifier(L"ProcGuard");
    notifier.Show(toast);
}
```

### 2. Advanced Toast with Actions
**XML Template:**
```xml
<toast launch="action=viewDetails&amp;blockId=123">
    <visual>
        <binding template="ToastGeneric">
            <image placement="appLogoOverride" src="C:\ProcGuard\icon.png"/>
            <text>Roblox Blocked</text>
            <text>Child attempted to launch Roblox at 11:47 PM</text>
        </binding>
    </visual>
    <actions>
        <action content="View Details" 
                arguments="action=view&amp;pid=1234" 
                activationType="foreground"/>
        <action content="Add 30 Minutes" 
                arguments="action=extend&amp;minutes=30" 
                activationType="background"/>
        <action content="Dismiss" 
                arguments="action=dismiss" 
                activationType="background"/>
    </actions>
    <audio src="ms-winsoundevent:Notification.Default"/>
</toast>
```

**CGO/C++ Implementation:**
```cpp
void ShowActionableToast(const wchar_t* title, const wchar_t* body, 
                         const wchar_t* imgPath) {
    // Load XML template
    std::wstring xmlTemplate = L"<toast>...</toast>";  // Full XML above
    auto xmlDoc = XmlDocument();
    xmlDoc.LoadXml(xmlTemplate);
    
    // Customize text
    auto textNodes = xmlDoc.GetElementsByTagName(L"text");
    textNodes.Item(0).InnerText(title);
    textNodes.Item(1).InnerText(body);
    
    // Set image
    auto imageNodes = xmlDoc.GetElementsByTagName(L"image");
    imageNodes.Item(0).Attributes().GetNamedItem(L"src").NodeValue(
        winrt::box_value(imgPath)
    );
    
    // Create and show
    auto toast = ToastNotification(xmlDoc);
    
    // Handle activation (when user clicks action button)
    toast.Activated([](ToastNotification const&, IInspectable const& args) {
        auto activation = args.as<ToastActivatedEventArgs>();
        std::wstring arguments = activation.Arguments().c_str();
        
        // Parse action (e.g., "action=extend&minutes=30")
        if (arguments.find(L"action=extend") != std::wstring::npos) {
            // Call Go callback to extend time
            NotifyGoExtendTime(30);
        }
    });
    
    auto notifier = ToastNotificationManager::CreateToastNotifier(L"ProcGuard");
    notifier.Show(toast);
}
```

### 3. Background Activation Handler
**For handling actions without opening UI**

**COM Registration Required:**
```xml
<!-- Package.appxmanifest -->
<Application>
    <Extensions>
        <desktop:Extension Category="windows.toastNotificationActivation">
            <desktop:ToastNotificationActivation 
                ToastActivatorCLSID="12345678-1234-1234-1234-123456789ABC"/>
        </desktop:Extension>
    </Extensions>
</Application>
```

**C++ COM Server:**
```cpp
class ToastActivator : public INotificationActivationCallback {
    HRESULT Activate(LPCWSTR appUserModelId, LPCWSTR invokedArgs, 
                     NOTIFICATION_USER_INPUT_DATA const* data, ULONG count) {
        // Parse invokedArgs (e.g., "action=extend&minutes=30")
        if (wcsstr(invokedArgs, L"action=extend")) {
            // Call ProcGuard service to extend time
            CallProcGuardService(L"ExtendTime", 30);
        }
        return S_OK;
    }
};
```

## Implementation Phases

### Phase 1: Basic Toasts (CGO + WinRT)
**Timeline:** 3-4 days
- [ ] Setup WinRT C++ project
- [ ] Implement basic text toasts
- [ ] Replace current web toasts
- [ ] Test on Windows 10 & 11

### Phase 2: Actionable Toasts (CGO + COM)
**Timeline:** 5-7 days
- [ ] Add action buttons to toasts
- [ ] Implement activation handler
- [ ] Wire up "Add 30 Minutes" action
- [ ] Test background activation

### Phase 3: Rich Media (CGO + WinRT)
**Timeline:** 2-3 days
- [ ] Add app icons to toasts
- [ ] Custom notification sounds
- [ ] Inline images (e.g., app screenshot)

### Phase 4: Notification Center Integration (Pure Go + CGO)
**Timeline:** 2-3 days
- [ ] Store notifications in database
- [ ] Show history in UI
- [ ] Sync with Windows Action Center

## Toast Types

### 1. Block Notification
```
[Icon: ⛔] Roblox Blocked
Child attempted to launch Roblox at 11:47 PM
[View Details] [Add 30 Minutes]
```

### 2. Time Limit Warning
```
[Icon: ⏰] Time Almost Up
Child has 5 minutes remaining for Minecraft
[Add Time] [View Usage]
```

### 3. Tamper Alert
```
[Icon: ⚠️] Tamper Attempt Detected
Child tried to rename chrome.exe to notchrome.exe
[View Details] [Dismiss]
```

### 4. Daily Summary
```
[Icon: 📊] Daily Summary
Child used computer for 3h 24m today
[View Report]
```

## CGO + Zig Benefits

### Why CGO is Needed:
1. **WinRT:** `Windows.UI.Notifications` requires C++/WinRT
2. **COM Registration:** Background activation needs COM server
3. **XML Manipulation:** `XmlDocument` easier in C++ than Go XML binding

### Zig Advantages:
- Static linking of `windowsapp.lib`
- Cleaner FFI for WinRT ABI
- Cross-compile notification tools

## Database Schema

```sql
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY,
    type TEXT,  -- 'block', 'warning', 'tamper', 'summary'
    title TEXT,
    message TEXT,
    icon_path TEXT,
    created_at TIMESTAMP,
    is_read BOOLEAN DEFAULT 0,
    action_taken TEXT  -- 'extended', 'dismissed', 'viewed'
);
```

## UI Integration

### Notification History
```svelte
<div class="notification-center">
  <h2>Recent Alerts</h2>
  {#each notifications as notif}
    <div class="notification-card" class:unread={!notif.is_read}>
      <img src={notif.icon_path} />
      <div>
        <h3>{notif.title}</h3>
        <p>{notif.message}</p>
        <span class="time">{formatTime(notif.created_at)}</span>
      </div>
    </div>
  {/each}
</div>
```

## Notification Preferences

### Settings UI
```
Notification Settings:
  [x] Block attempts
  [x] Time limit warnings
  [ ] Daily summaries
  [x] Tamper alerts
  
Sound: [Dropdown: Default / Silent / Custom]
Priority: [Dropdown: Normal / High]
Show on Lock Screen: [Toggle]
```

## Testing Plan

### Test Cases:
1. **Basic Toast:** Trigger block, verify toast appears
2. **Action Button:** Click "Add 30 Minutes", verify time extended
3. **Background:** Close UI, verify toast still works
4. **Focus Assist:** Enable Do Not Disturb, verify priority toasts shown
5. **Multi-Monitor:** Verify toast shows on correct monitor

### Success Metrics:
- [ ] Toasts appear within 1 second of event
- [ ] Action buttons work 100% of time
- [ ] No UI required for toast display
- [ ] Notifications persist in Action Center

## Privacy Considerations
- Toasts show on lock screen (optional, can disable)
- Notification history stored locally only
- Option to auto-clear old notifications

## Future Enhancements
- **Mobile App:** Push notifications to parent's phone (via cloud)
- **Scheduled Toasts:** Reminder: "Check on child's screen time"
- **AI Summaries:** Weekly digest with insights
