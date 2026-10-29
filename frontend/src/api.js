/**
 * Thin wrapper around the canteen backend HTTP API.
 * All paths are relative so the Vite proxy handles host/port.
 *
 * Customer and admin sessions are stored separately: being signed in as a
 * customer never grants the dashboard. Requests use the session for the
 * current route, and /admin always uses the admin session.
 */

const SESSION_PREFIX = 'canteen.session.';
const CREDS_PREFIX = 'canteen.creds.';

function read(key) {
  try { return JSON.parse(localStorage.getItem(key)); } catch { return null; }
}

function write(key, value) {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, JSON.stringify(value));
  } catch { /* private mode / quota */ }
}

const sessions = {
  customer: read(SESSION_PREFIX + 'customer'),
  admin: read(SESSION_PREFIX + 'admin'),
};
const credentials = {
  customer: read(CREDS_PREFIX + 'customer'),
  admin: read(CREDS_PREFIX + 'admin'),
};

export const routeScope = () =>
  window.location.pathname.startsWith('/admin') ? 'admin' : 'customer';

export const auth = {
  get scope() { return routeScope(); },
  get token() { return sessions[routeScope()]?.token ?? null; },
  get role() { return sessions[routeScope()]?.role ?? null; },
  get isAdmin() { return this.role === 'admin'; },

  session(scope = routeScope()) { return sessions[scope]; },
  hasSession(scope = routeScope()) { return !!sessions[scope]?.token; },
  savedCredentials(scope = routeScope()) { return credentials[scope]; },

  async login(principal, password, scope = routeScope()) {
    const res = await api('/auth/login', {
      method: 'POST',
      body: { principal, password },
      noAuth: true,
    });
    sessions[scope] = { token: res.token, role: res.role, principal };
    write(SESSION_PREFIX + scope, sessions[scope]);
    // Cached so the login form is prefilled next time on this device.
    credentials[scope] = { principal, password };
    write(CREDS_PREFIX + scope, credentials[scope]);
    return res;
  },

  // Logging in as an admin from the storefront moves the fresh token into the
  // admin slot instead of leaving an admin token usable by the customer app.
  promoteToAdmin(from = routeScope()) {
    if (from === 'admin') return;
    sessions.admin = sessions[from];
    write(SESSION_PREFIX + 'admin', sessions.admin);
    credentials.admin = credentials[from];
    write(CREDS_PREFIX + 'admin', credentials.admin);
    credentials[from] = null;
    write(CREDS_PREFIX + from, null);
    sessions[from] = null;
    write(SESSION_PREFIX + from, null);
  },

  logout(scope = routeScope()) {
    sessions[scope] = null;
    write(SESSION_PREFIX + scope, null);
  },

  clearCredentials(scope = routeScope()) {
    credentials[scope] = null;
    write(CREDS_PREFIX + scope, null);
  },
};

/**
 * Internal fetch helper. Attaches the route's JWT and serialises the body.
 */
async function api(path, { method = 'GET', body, noAuth = false, query, scope } = {}) {
  const url = new URL(path, window.location.origin);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== '') url.searchParams.set(k, v);
    }
  }

  const active = scope ?? routeScope();
  const headers = { 'Content-Type': 'application/json' };
  if (!noAuth && sessions[active]?.token) {
    headers['Authorization'] = `Bearer ${sessions[active].token}`;
  }

  const res = await fetch(url.toString(), {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { const j = await res.json(); msg = j.error || j.message || msg; } catch {}
    throw new Error(msg);
  }

  if (res.status === 204) return null;
  return res.json();
}

// ── Auth ────────────────────────────────────────────────────────────────────
export const getMe = (scope) => api('/auth/me', { scope });

// ── Users ───────────────────────────────────────────────────────────────────
export const getUsers     = ()          => api('/users');
export const createUser   = (body)      => api('/users', { method: 'POST', body });
export const topUp        = (id, paise) => api(`/users/${id}/topup`, { method: 'POST', body: { amount_paise: paise } });
export const getWallet    = (userId)    => api('/wallet', { query: { user_id: userId } });

// ── Menu ────────────────────────────────────────────────────────────────────
export const getMenu       = (all = false) => api('/menu', { query: { all: all ? 1 : undefined } });
export const createMenuItem = (body)       => api('/menu', { method: 'POST', body });
export const updateMenuItem = (id, body)   => api(`/menu/${id}`, { method: 'PUT', body });
export const patchMenuItem  = (id, body)   => api(`/menu/${id}/availability`, { method: 'PATCH', body });
export const deleteMenuItem = (id)         => api(`/menu/${id}`, { method: 'DELETE' });

// ── Orders ──────────────────────────────────────────────────────────────────
// Admins must name the customer; a customer token overrides this server-side.
export const createOrder = (userId, lines) =>
  api('/orders', { method: 'POST', body: { user_id: Number(userId), lines } });
export const listOrders  = (query = {})  => api('/orders', { query });
export const getOrder    = (id)          => api(`/orders/${id}`);
export const cancelOrder = (id)          => api(`/orders/${id}/cancel`, { method: 'POST' });

// ── Queue ───────────────────────────────────────────────────────────────────
export const getQueue = (lane) => api('/queue', { query: { lane } });
export const ackQueue = (lane, id) =>
  api(`/queue/${lane}/ack`, { method: 'POST', body: { id } });
