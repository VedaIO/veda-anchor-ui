import { writable } from 'svelte/store';

export const isExtensionInstalled = writable<boolean | null>(null);

let pollingInterval: any;

export async function checkExtension(): Promise<void> {
  // Initial check
  await performCheck();

  // Start polling if not installed
  // This handles the case where the extension connects later (Native Messaging)
  if (pollingInterval) clearInterval(pollingInterval);
  pollingInterval = setInterval(async () => {
    const installed = await performCheck();
    if (installed) {
      clearInterval(pollingInterval);
    }
  }, 2000); // Check every 2 seconds
}

async function performCheck(): Promise<boolean> {
  try {
    // Use Wails backend method to check if extension is installed
    const installed = await window.go.main.App.CheckChromeExtension();
    console.log('Extension check result:', installed);

    if (installed) {
      isExtensionInstalled.set(true);

      // Register with backend
      // We register both the Store ID and the Dev ID to be safe
      const storeExtensionId = 'hkanepohpflociaodcicmmfbdaohpceo';
      const devExtensionId = 'gpaafgcbiejjpfdgmjglehboafdicdjb';

      try {
        await window.go.main.App.RegisterExtension(storeExtensionId);
        await window.go.main.App.RegisterExtension(devExtensionId);
        console.log('Extensions registered with backend');
      } catch (err) {
        console.error('Failed to register extension:', err);
      }
      return true;
    } else {
      isExtensionInstalled.set(false);
      return false;
    }
  } catch (error) {
    console.error('Error checking extension:', error);
    isExtensionInstalled.set(false);
    return false;
  }
}

// Listen for extension connection event from backend
// We expose this setup function to be called from onMount
export function setupExtensionListener() {
  if (window.runtime) {
    window.runtime.EventsOn('extension_connected', (connected: boolean) => {
      console.log('Extension connected event received:', connected);
      if (connected) {
        isExtensionInstalled.set(true);
        if (pollingInterval) clearInterval(pollingInterval);
      }
    });
  }
}
