// Short vibration on supported devices; a no-op elsewhere (including iOS).
export function haptic(ms = 8) {
  try { navigator.vibrate?.(ms); } catch { /* unsupported */ }
}

// Pattern vibrations (on/off durations in ms), e.g. [140, 90, 140].
export function hapticPattern(pattern) {
  try { navigator.vibrate?.(pattern); } catch { /* unsupported */ }
}
