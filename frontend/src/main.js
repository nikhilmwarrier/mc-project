import { mount } from 'svelte';
import App from './App.svelte';
import './style.css';

if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {});
  });
}

mount(App, { target: document.getElementById('app') });
