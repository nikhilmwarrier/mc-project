# Distributed IoT Canteen Queue Management System

Low-cost, high-throughput queue system for a busy campus canteen. Standard meals
(85%) flow down a rapid line that never touches the kitchen; short orders (15%)
are trayed in the kitchen and picked up at a separate counter. Three ESP32
stations display tokens, chime, and confirm handovers over MQTT; one Go backend
owns money and queue state.

```
kiosk/curl ──HTTP──▶ backend ──MQTT──▶ Node 1 kitchen ──tray──▶ Node 3 counter B
 (orders)      │         │                    Node 2 counter A (rapid meals)
               │         └── acks ◀── buttons on all three nodes
               └── wallet, menu, admin
```

## Layout

| Path | What |
|---|---|
| `canteen-node/` | The one ESP32 sketch for all three nodes (`ROLE` flag picks kitchen / counter-A / counter-B) |
| `backend/` | Go backend: wallet + orders + HTTP API + MQTT queue service (subtree import, full history kept) |
| `mqtt/` | Bench broker config + `tui/`, the Go test rig that stands in for the backend |
| `justfile` | Shortcuts: `just a/b/kitchen` flash, `just build`, `just test` |
| `architecture.md` | Source of truth: flows, topics, APIs, pins, build order |
| `pinout.md` | Wiring for all three (identical) nodes |
| `BOM.md`, `plan-summary.md` | Parts list, original idea draft |

## Bring-up order

1. **Broker:** `mosquitto -c mqtt/mosquitto.conf` (LAN-open, lab only).
2. **Fake backend:** `cd mqtt/tui && go run . -broker 127.0.0.1` — owns the queues, advances on button acks. Keys: `1/2/3` enqueue, `a/k/b` simulate a button, `q` quit.
3. **Flash nodes:** `just a`, `just b`, `just kitchen` — stamps the role and this laptop's current LAN IP as `MQTT_HOST`, then uploads (default port `/dev/cu.usbmodem101`, override with `PORT=...`).
4. **Real backend + UI:** `just serve` — builds the frontend, then serves the app and API together on `:8080` and prints a QR code for `https://<lan-ip>:8080` (or `http://` before TLS is set up). Scan it from a phone on the same Wi-Fi. Same server handles `/` (storefront) and `/admin` (staff dashboard); sign in with the bootstrapped admin or a customer’s roll number + PIN.

   For HTTPS (needed for PWA install and notifications on a phone): `just tls` generates a certificate for the current LAN IP with mkcert. Install the printed root CA (`~/Library/Application Support/mkcert/rootCA.pem`) on the phone, then restart `just serve`. Re-run `just tls` after the laptop’s IP changes.
5. **UI development:** `cd frontend && pnpm dev` — Vite on `:5173` with hot reload, proxying `/auth`, `/users`, `/menu`, `/orders`, `/queue` to the backend on `:8080`.

## Hardware (per node, ~₹2,000 total for three)

ESP32-C3 + 16x2 I2C LCD (`0x27`/`0x3F` — scan first) + HYDZ 2-pin piezo (`+`→GPIO7, `-`→GND) + button (GPIO10→GND, `INPUT_PULLUP`). I2C on GPIO5/6. Never GPIO8/9 (strapping) or 18/19 (USB). Boards ship with unsoldered headers — solder before breadboarding; USB adapters/cables are not in the BOM, procure separately.

## Firmware notes (earned the hard way)

- One sketch, no `delay()` anywhere; Wi-Fi/MQTT reconnect on timers, LCD writes only on change.
- `WiFi.begin()` is called **exactly once** — re-calling it mid-handshake wedges the driver (`cannot set config`).
- Two workarounds for weak-transmit board batches, both in the sketch: 11b-only protocol (`esp_wifi_set_protocol`) and `WiFi.setTxPower(WIFI_POWER_8_5dBm)` after `begin()` (see espressif/arduino-esp32#6767). If a node scans fine but never joins with reason 2, this is why.
- Phone hotspots reassign the laptop's IP regularly — when the smoke log goes quiet while buttons still beep, recheck `MQTT_HOST` against `ipconfig getifaddr en0`.

## Backend notes

- All money in paise (`int64`), never floats. Balances debit atomically inside order creation.
- Token = `orders.id`. `pending` means paid but not handed over; only a serve/collect ack completes an order (duplicates dedupe via `ErrConflict`).
- Payloads are plain ASCII for the 16-char LCDs (`105 Q44`, `106 POR Q3`), QoS 1, display topics retained.
- `backend/` is subtree-managed from the backend repo — don't hand-edit around merges; tooling lives outside it. Two Go modules (`backend/food-kiosk`, `mqtt/tui`) stay separate on purpose; `just build`/`just test` cover both.
