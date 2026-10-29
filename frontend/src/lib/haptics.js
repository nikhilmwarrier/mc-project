// Short vibration on supported devices; a no-op elsewhere (including iOS).
export function haptic(ms = 8) {
  try { navigator.vibrate?.(ms); } catch { /* unsupported */ }
}
