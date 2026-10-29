/**
 * Thin wrapper around the canteen backend HTTP API.
 * All paths are relative so the Vite proxy handles host/port.
 */

let _token = null;
let _role = null;

// Survive a reload or direct navigation to /admin.
try {
  _token = sessionStorage.getItem('canteen_token');
  _role = sessionStorage.getItem('canteen_role');
} catch { /* private mode */ }

export const auth = {
  get token() { return _token; },
  get role()  { return _role; },
  get isAdmin() { return _role === 'admin'; },

  async login(principal, password) {
    const res = await api('/auth/login', {
      method: 'POST',
      body: { principal, password },
      noAuth: true,
    });
    _token = res.token;
    _role  = res.role;
    try {
      sessionStorage.setItem('canteen_token', _token);
      sessionStorage.setItem('canteen_role', _role);
    } catch { /* private mode */ }
    return res;
  },

  logout() {
    _token = null;
    _role = null;
    try {
      sessionStorage.removeItem('canteen_token');
      sessionStorage.removeItem('canteen_role');
    } catch { /* private mode */ }
  },
};

/**
 * Internal fetch helper. Automatically attaches the JWT and serialises body.
 */
async function api(path, { method = 'GET', body, noAuth = false, query } = {}) {
  const url = new URL(path, window.location.origin);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== '') url.searchParams.set(k, v);
    }
  }

  const headers = { 'Content-Type': 'application/json' };
  if (!noAuth && _token) headers['Authorization'] = `Bearer ${_token}`;

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
export const getMe = () => api('/auth/me');

// ── Users ───────────────────────────────────────────────────────────────────
export const createUser = (body)      => api('/users', { method: 'POST', body });
export const topUp      = (id, paise) => api(`/users/${id}/topup`, { method: 'POST', body: { amount_paise: paise } });
export const getWallet  = (userId)    => api('/wallet', { query: { user_id: userId } });

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
