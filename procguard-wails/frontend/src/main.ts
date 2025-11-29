import '~bootstrap/dist/js/bootstrap.bundle.min.js';
import './global.css';
import App from './App.svelte';
import * as runtime from '../wailsjs/runtime';

// Wait for the DOM to be fully loaded before doing anything.
document.addEventListener('DOMContentLoaded', () => {
  // Now that the DOM is ready, wait for the Wails runtime to be ready.
  runtime.EventsOn('wails:ready', () => {
    // Now we are sure that both the DOM and the Wails backend are ready.
    const app = new App({
      target: document.getElementById('app'),
    });
  });
});
