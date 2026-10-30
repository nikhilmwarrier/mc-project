---
title: Project Architecture
date: 2026-10-30
---

## 1. The project

A low-cost queue system for a busy campus canteen. Two kinds of service stay apart:

- The **standard meal line** (about 85% of orders) is fast: rice, curries, sides,
  served in 5–10 seconds each. The UI calls this **Counter 1**; the firmware and
  MQTT topics still use the original `counter-a` name.
- **Short orders** (about 15%) — porotta, chicken curry, snacks — are assembled in
  the **kitchen** and collected at **Counter 2** (firmware: `counter-b`).

Three ESP32-C3 stations display tokens, chime, and confirm handovers over MQTT. One
Go backend owns money, orders, and the queues, and it also serves the web app.

### Hardware

- 3× ESP32-C3 development boards
- 3× 16×2 LCDs with I2C backpack (`0x27` or `0x3F`)
- 3× HYDZ 5V active piezo buzzers (2-pin)
- Tactile pushbuttons (`INPUT_PULLUP`, software debounced)
- Breadboards, jumpers, 3× 5V USB adapters + cables

The USB adapters/cables are not in the BOM yet, and the C3 boards ship with
**unsoldered** headers — solder them before breadboarding.

Everything runs on one local Wi-Fi router and a local Mosquitto broker. No cloud.

### The three stations

- **Node 1 — Kitchen.** Shows incoming short orders and their item codes; staff press
  once the tray is packed. This moves the token to Counter 2's ready lane.
- **Node 2 — Counter 1 (meals).** Shows who is being served and the queue depth;
  staff press as each customer is handed over. Never touches the kitchen.
- **Node 3 — Counter 2 (specials pickup).** Lights up when a tray is ready, chimes,
  and waits for staff to confirm collection.

### Design rules

- **Token = `orders.id`.** No separate daily counter.
- **`pending` means paid but not handed over.** Neither publishing nor “ready”
  completes an order; only the serve/collect ack does.
- **The backend owns every queue.** The nodes are thin displays with one button.
- **Money is paise (`int64`)**, debited atomically in `CreateOrder` before MQTT.
- **One source of truth for lanes:** HTTP, MQTT, and the web UI all read the same
  `backend.Service`, never a parallel query.

---

## 2. Repository and runtime layout

```
mc-project/
├── backend/                  # Go backend (subtree import, full history)
│   └── food-kiosk/
│       ├── main.go           # config → DB → MQTT + HTTP + SPA + QR
│       ├── database/         # users, menu, orders, ledger, schema
│       ├── server/           # HTTP API, auth, static SPA serving
│       ├── backend/          # fulfillment lanes (a / kitchen / ready)
│       ├── mqtt/             # paho client: retained displays + acks
│       ├── broker/           # deploy-side Mosquitto config
│       └── api_contract.yaml
├── frontend/                 # Svelte 5 + Vite SPA (customer + admin routes)
├── canteen-node/             # one ESP32 sketch, three roles
├── mqtt/                     # bench broker config + Go TUI test rig
├── justfile                  # flash / serve / build / test shortcuts
├── architecture.md           # this file
├── pinout.md                 # wiring for all nodes
├── README.md, BOM.md
```

Two Go modules stay separate on purpose (backend is subtree-managed; the TUI is
tooling): `backend/food-kiosk` and `mqtt/tui`. `just build` / `just test` cover both.

---

## 3. Order flows

### Placing an order

`POST /orders` → `CreateOrder()` writes order, lines, and ledger debit in one
transaction. Routing is by menu code:

- **only `MEAL`** → Counter 1 queue (`lane = a`).
- **anything else** → kitchen queue (`lane = kitchen`). Non-meal codes are joined
  for the LCD, e.g. `POR/CHK`.
- Mixed carts go to the kitchen as one tray/token.

The queue service (`backend/`) stores classification in a `fulfillment` row on first
reconcile; it is frozen for the order's lifetime.

### Counter 1 (meals)

1. Backend publishes retained `canteen/counter-a/now` = `<head> Q<depth>`.
2. Node 2 shows `NOW:<id>` / `Q:<n>`, beeps.
3. Staff press → `canteen/counter-a/served <id>`.
4. Backend calls `CompleteOrder(id)`, republishes the next head.
5. A duplicate ack (QoS 1 retry) is treated as already done and only refreshes.

### Counter 2 (specials)

1. Backend publishes retained `canteen/kitchen/now` = `<id> <codes> Q<depth>`.
2. Node 1 shows `TK:<id> [<codes>]` / `Q:<n>`.
3. Staff press → `canteen/kitchen/done <id>`.
4. Backend moves the fulfillment row to `ready` and publishes retained
   `canteen/counter-b/ready` = `<id> <codes> W<n>`. **Not completed yet.**
5. Node 3 shows `RDY:<id> [<codes>]` / `W:<n>` and chimes.
6. Staff press → `canteen/counter-b/collected <id>`.
7. Backend completes the order, removes it from ready, publishes the next.

### Customer notifications

The storefront polls each tracked order's detail every 4s:

- **Counter 1 orders** are shown immediately after ordering (pending until served).
- **Counter 2 orders** show a bottom sheet only when `ready = true`; the phone
  vibrates in a three-beep pattern once and a browser notification fires (if
  permitted). Dismissing leaves a `Show token (N)` FAB; tapping reopens the sheet.
- Once the order is completed or cancelled it disappears from tracking.

### Lunch rush

45 meals ordered between 11:55–12:05 sit as `pending`; the LCD shows only the head
and depth. At 5–10s per customer the queue drains in ~4–7 minutes. Retained display
messages restore a rebooted node's view; duplicate acks are idempotent.

---

## 4. Backend

Go 1.27, SQLite via `modernc.org/sqlite` (no cgo). All money is `int64` paise.

### Packages

- **`database/`** — schema, users, menu, orders, order items, transactions.
  `CreateOrder` is atomic (order + lines + debit); `CancelOrder` refunds in the same
  transaction; sentinel errors (`ErrNotFound`, `ErrInvalidInput`, `ErrConflict`,
  `ErrInsufficientBalance`). Snapshot pricing in `order_items`.
- **`backend/`** — fulfillment state. Creates an additive `fulfillment` table at
  startup (does not change `user_version`) with `order_id`, `lane`
  (`a|kitchen|ready`), `code`, `ready_sequence`. `Snapshot`, `Queue`, `IsReady`,
  `Acknowledge`. `reconcile` classifies pending orders missing a row, so orders
  written by older code or before a crash still appear.
- **`server/`** — HTTP API, JWT auth, rate-limited login, admin bootstrap, SPA
  static hosting.
- **`mqtt/`** — paho client. Publishes retained display payloads, consumes acks.

### HTTP server and static app

`main.go` wires: env config → SQLite → admin bootstrap → queue service → MQTT
goroutine → HTTP server. One Go process serves both the API and the built SPA.

- `WEB_DIR` (default `../../frontend/dist`) is served at `/` with SPA fallback, so
  `/admin` survives a hard refresh. API patterns win over the catch-all.
- Missing `index.html` = API-only mode with a startup warning.
- On startup it prints the LAN URL and a terminal **QR code**. The IP is chosen from
  `en0`, then `en1`, then the default route — the same preference as `just a/b/kitchen`
  uses for `MQTT_HOST`, so the QR and the firmware agree.
  `PUBLIC_HOST` overrides the QR host; `ADDR` changes the port (default `:8080`).
- **HTTPS:** set `TLS_CERT` + `TLS_KEY` (both or neither) and the server speaks TLS;
  the QR switches to `https://`. `just tls` generates a locally-trusted cert for the
  current LAN IP with mkcert. The phone must trust mkcert’s root CA for install/PWA
  and notifications to work. Certs live in `certs/` (gitignored); regenerate after an
  IP change.
- `just serve` builds the frontend and starts this server.

Config (env): `ADDR`, `DB_PATH`, `JWT_SECRET`, `JWT_TTL`, `AUTH_REQUIRED`,
`CORS_ORIGINS`, `ADMIN_PRINCIPAL`, `ADMIN_PASSWORD`, `LOGIN_MAX_ATTEMPTS`,
`LOGIN_WINDOW`, `MQTT_BROKER`, `MQTT_CLIENT_ID`, `MQTT_USERNAME`, `MQTT_PASSWORD`,
`WEB_DIR`, `PUBLIC_HOST`.

### Auth

- Customers sign in with roll number + PIN; admins with a bootstrapped principal.
- Passwords/PINs are bcrypt hashes in a `credentials` table; JWTs are HS256 with
  role and user id. `AUTH_REQUIRED=false` exists only for curl demos and trusts the
  request's `user_id`; never use it on an untrusted network.
- A customer token always overrides a body `user_id`; an admin must name the
  customer.
- The storefront and dashboard keep **separate sessions**: customer token in
  `canteen.session.customer`, admin token in `canteen.session.admin`. Visiting
  `/admin` as a customer shows the admin login; logging in there does not disturb
  the customer session. Admin login from `/` promotes the token and redirects.
- Login forms cache principal + password in `localStorage` per scope to prefill next
  time. This is plaintext on the device — a deliberate convenience trade-off.

---

## 5. HTTP API

All paths below are implemented and tested.

```
GET  /healthz

POST /auth/login                 {principal, password} -> {token, role}
GET  /auth/me                    -> {id, roll_no, name, phone, role}

POST /users                      {roll_no, name, phone, pin?} -> {id}
GET  /users                      -> {users:[{id, roll_no, name}]}     (admin)
POST /users/:id/topup            {amount_paise} -> {balance_paise}
GET  /wallet                     ?user_id= -> {balance_paise}          (self/admin)

GET  /menu                       ?all=1 (admin) -> {items:[...]}
POST /menu                       (admin)
PUT  /menu/:id                   (admin)
PATCH /menu/:id/availability     {is_available} (admin)
DELETE /menu/:id                 (admin; 409 if ordered)

POST /orders                     {user_id, lines:[{code, quantity}]}
                                 -> {id, total_paise}
GET  /orders                     ?status=&user_id= -> {orders:[...]}
GET  /orders/:id                 -> order + items + ready:bool
POST /orders/:id/cancel          -> {balance_paise}

GET  /queue                      ?lane=a|kitchen|ready
                                 -> {head, depth, items:[ids]}
POST /queue/:lane/ack            {id} -> 204                           (admin)

GET  /                           SPA (storefront)
GET  /admin                      SPA (dashboard)
```

Notes:

- `GET /queue` reports the same fulfillment lanes the MQTT client publishes; the
  nodes never call it. `depth` is the number of entries (includes the head).
- `POST /queue/:lane/ack` is the HTTP equivalent of a station button. It is **not**
  a cancel: `POST /orders/:id/cancel` refunds and keeps order history.
- `GET /orders/:id` carries `ready`, which drives the customer's ready sheet.
- `GET /orders` with no filter returns all orders for an admin, a customer's own
  history otherwise. `MENU` writes are admin-only; `GET /menu?all=1` is admin-only.

---

## 6. MQTT contract

Local Mosquitto, QoS 1, retained display topics, plain ASCII.

| Topic | Direction | Retained | Payload | Example |
|---|---|---|---|---|
| `canteen/counter-a/now` | backend → Node 2 | yes | `<head> Q<depth>` / `IDLE Q0` | `105 Q44` |
| `canteen/counter-a/served` | Node 2 → backend | no | `<id>` | `105` |
| `canteen/kitchen/now` | backend → Node 1 | yes | `<id> <codes> Q<depth>` / `IDLE Q0` | `106 POR/CHK Q3` |
| `canteen/kitchen/done` | Node 1 → backend | no | `<id>` | `106` |
| `canteen/counter-b/ready` | backend → Node 3 | yes | `<id> <codes> W<n>` / `NONE W0` | `106 POR/CHK W2` |
| `canteen/counter-b/collected` | Node 3 → backend | no | `<id>` | `106` |
| `canteen/<node>/online` | node → all (LWT) | yes | `1` / `0` | `1` |

- Display payloads are republished on a 1s reconciliation tick only when changed.
- `Q<n>` on Counter 1/kitchen excludes the shown head; `W<n>` on ready includes it.
- Item codes are joined with `/`, sanitised, and capped; the 16-char LCD truncates
  overflow. Mixed orders show all non-meal codes.
- Ack payloads must be canonical decimal ids. Retained/duplicate acks are ignored or
  treated as already applied. `ErrUnexpectedAck`/`ErrInvalidAck` trigger a
  republish; transient DB errors leave the message unacked so the broker redelivers.
- Client uses persistent session, auto-reconnect, keepalive 20s, ordered dispatch
  off, automatic ack off.

A fake backend with the same protocol lives in `mqtt/tui`:

```
cd mqtt/tui && go run . -broker 127.0.0.1
# 1/2/3 enqueue, a/k/b simulate a button, q quit
```

It prints lane state, node presence from LWT, and validates acks against the head.

---

## 7. Frontend

Svelte 5 (runes) + Vite. No router dependency: a small history-API helper switches
between `/` and `/admin`; unknown paths normalise to `/`.

### Routes and sessions

- `/` — customer storefront. Requires a customer session; admin sessions are
  separate and redirect to `/admin`.
- `/admin` — staff dashboard. Requires an **admin** session; customers see a fresh
  admin login, with their customer session preserved.
- Tokens are scoped by route (`routeScope()`), stored in `localStorage`; login
  credentials are cached per scope for prefill.

### Storefront

- Menu from `GET /menu`, split into **Counter 1** (code `MEAL`) and **Counter 2**
  (everything else). Real food photos for `MEAL`, `POR`, `CHP`, `CHK`; Lucide
  fallback otherwise.
- Wallet balance in the header, cart sheet with quantity steppers, total, wallet
  check, and atomic `POST /orders` checkout.
- Order tracking: ids in `sessionStorage`; poll `GET /orders/:id` every 4s.
  - Counter 1 tokens appear immediately in the token sheet/FAB until acked at
    Counter 1.
  - Counter 2 tokens open the ready sheet on transition, vibrate
    `[140,90,140,90,140]`, and fire a browser notification if granted. Dismiss →
    `Show token (N)` FAB.
- Installable PWA: manifest, 192/512/maskable icons, minimal service worker
  (install/claim, network passthrough — no offline cache), install button when
  `beforeinstallprompt` fires.

### Dashboard

Tabs: Queues, Orders, Menu, Users. User-ID entry is replaced by user dropdowns
(`GET /users`), order filters, top-up, and wallet. Queues show head/depth/items and
an Ack button that calls the ack endpoint (not cancel). Activity appears as toasts.

### Design system

- Body: self-hosted **IBM Plex Sans**; logo: **Boogaloo**, reading “College Canteen”.
- **Lucide** icons only; no emoji in the UI.
- Light theme, white surfaces, hairline dividers; menu rows (not cards) with image
  right and a floating ADD/stepper straddling the image edge.
- Shared easing tokens, press-scale feedback, `@starting-style` sheet/cart entries,
  reduced-motion support, and `navigator.vibrate` haptics where appropriate.
- Food photo credits: `frontend/public/food/CREDITS.md`.

### Development

`cd frontend && pnpm dev` runs Vite at `:5173` proxying `/auth`, `/users`, `/wallet`,
`/menu`, `/orders`, `/queue` to the backend at `:8080`. Production is served by the
Go server (`just serve`).

---

## 8. Firmware

One sketch (`canteen-node/canteen-node.ino`) for all nodes; role is compile-time:

```cpp
#define ROLE ROLE_KITCHEN   // or ROLE_COUNTER_A / ROLE_COUNTER_B
```

Libraries: `LiquidCrystal_I2C`, `PubSubClient`. Debounce (~30ms) and beeps (150ms)
are `millis()` code, no extra libraries.

### Pins (ESP32-C3)

| Signal | Pin |
|---|---|
| LCD SDA | GPIO5 |
| LCD SCL | GPIO6 |
| Buzzer `+` | GPIO7 (2-pin piezo, `-` to GND; no transistor) |
| Button | GPIO10 (`INPUT_PULLUP`, other leg GND) |

Avoid GPIO8/9 (strapping) and GPIO18/19 (USB-CDC). Scan I2C first (`0x27` vs
`0x3F`).

### Loop and Wi-Fi

`loop()` never sleeps: Wi-Fi maintenance → MQTT reconnect + `client.loop()` →
debounced button → non-blocking beep. LCD writes only when content changes.

Hard-won details, all present in the sketch:

- `WiFi.begin()` is called **once**; re-calling it mid-handshake wedges the driver
  (`sta is connecting, cannot set config`).
- `WiFi.setSleep(false)`.
- **DSSS-only (`esp_wifi_set_protocol(WIFI_PROTOCOL_11B)`)** and
  `WiFi.setTxPower(WIFI_POWER_8_5dBm)` after `begin()` for the weak-transmit C3
  batch (espressif/arduino-esp32#6767).
- 15s heartbeat logging when down; never touch the driver in the retry loop.
- LWT on `canteen/<node>/online`; retained `1` when connected.

### Roles

| Role | Subscribes | Button publishes | LCD |
|---|---|---|---|
| Kitchen | `kitchen/now` | `kitchen/done <shown>`, clears | `TK:106 [POR/CHK]` / `Q:3` |
| Counter A | `counter-a/now` | `counter-a/served <shown>` | `NOW:105` / `Q:44` |
| Counter B | `counter-b/ready` | `counter-b/collected <shown>`, clears | `RDY:106 [POR]` / `W:2` |

Beep on new messages and presses; empty queues show `IDLE`/`NONE`, button no-op.
Bring-up order: Counter B → Kitchen → Counter A.

### Flashing

`just a`, `just b`, `just kitchen` fetch the laptop's `en0`/`en1` IPv4, stamp it as
`MQTT_HOST`, set the role, upload via `arduino-cli`, then restore the sketch.
Override the port with `PORT=/dev/cu.usbmodemXXX`. This is the same IP printed in
the server's QR code.

---

## 9. Tooling

```
just serve      # build frontend, run backend+API+SPA, print LAN QR (HTTPS if certs exist)
just tls        # mkcert cert for this LAN IP
just backend    # Go server only (serves the last frontend build)
just ui         # Vite dev server with proxy
just mqtt       # Mosquitto with the bench config
just a|b|kitchen# flash a node
just build      # build both Go modules
just test       # test backend + TUI
```

Backend tests cover money/overflow, ledger refunds, auth, routing, ack idempotency,
and queue recovery; the TUI has tests for advance/format. The frontend is validated
with `pnpm build` and browser flows.

---

## 10. Build order

1. Broker + backend MQTT; watch `canteen/#` with `mosquitto_sub`.
2. One node at a time: Counter B → Kitchen → Counter A.
3. End-to-end: place order → node LCD → button → next station → balance/order state.
4. `just serve`; scan the QR from a phone on the same Wi-Fi; verify storefront,
   admin login, wallet, ready sheet/FAB.
5. Resilience: kill the router mid-queue; nodes reconnect, retained displays
   restore, duplicate acks do not double-apply.

### Deliberately out of scope

- Veg/non-veg flags, ratings, and item descriptions — no data in the menu schema.
  Item cards show only name, price, and photo.
- Offline caching / Web Push — the service worker is install-only; notifications
  require the page to be open. PWA install and notifications need HTTPS (or
  localhost) with the mkcert CA trusted on the phone.
- Daily-resetting tokens (uses `orders.id`), cloud broker, OTA, native apps.
- HTTP beyond the routes in section 5.
