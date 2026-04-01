// ── Auth state ────────────────────────────────────────────────────────────────
export function getToken() {
  return localStorage.getItem('token');
}

export function getUser() {
  const raw = localStorage.getItem('user');
  if (!raw) return null;
  try { return JSON.parse(raw); } catch { return null; }
}

export function saveAuth(token, user) {
  localStorage.setItem('token', token);
  localStorage.setItem('user', JSON.stringify(user));
}

export function clearAuth() {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
}

// ── Circle state ──────────────────────────────────────────────────────────────
export function getActiveCircleId() {
  return localStorage.getItem('activeCircleId');
}

export function setActiveCircleId(id) {
  localStorage.setItem('activeCircleId', id);
}

// ── Server URL ────────────────────────────────────────────────────────────────
export function getServerUrl() {
  return localStorage.getItem('serverUrl') || window.location.origin;
}

export function setServerUrl(url) {
  localStorage.setItem('serverUrl', url);
}

// ── API client ────────────────────────────────────────────────────────────────
export async function api(method, path, body) {
  const base = getServerUrl();
  const token = getToken();
  const opts = {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  };
  if (body !== undefined) {
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(`${base}${path}`, opts);
  if (res.status === 401) {
    clearAuth();
    navigate('login');
    throw new Error('Unauthorized');
  }
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const data = await res.json();
      msg = data.error || data.message || msg;
    } catch { /* ignore */ }
    throw new Error(msg);
  }
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) {
    return res.json();
  }
  return null;
}

// ── Router ────────────────────────────────────────────────────────────────────
const routes = {};

export function registerRoute(name, renderFn) {
  routes[name] = renderFn;
}

export function navigate(route) {
  window.location.hash = route;
}

const AUTHED_ROUTES = ['map', 'history', 'places', 'circle', 'profile'];

async function render() {
  const hash = window.location.hash.replace('#', '') || 'map';
  const content = document.getElementById('content');
  const nav = document.getElementById('nav');

  // Auth guard
  if (AUTHED_ROUTES.includes(hash) && !getToken()) {
    navigate('login');
    return;
  }

  // Show/hide nav
  if (hash === 'login' || hash === 'register') {
    nav.style.display = 'none';
  } else {
    nav.style.display = 'flex';
  }

  // Update active tab
  document.querySelectorAll('.nav-link').forEach(link => {
    link.classList.toggle('active', link.dataset.route === hash);
  });

  // Initialize active circle if authenticated and not set
  if (getToken() && !getActiveCircleId()) {
    try {
      const circles = await api('GET', '/circles');
      if (Array.isArray(circles) && circles.length > 0) {
        setActiveCircleId(circles[0].id);
      }
    } catch { /* ignore */ }
  }

  // Call registered route
  const fn = routes[hash];
  if (fn) {
    // Clear content safely
    while (content.firstChild) content.removeChild(content.firstChild);
    await fn(content);
  } else {
    while (content.firstChild) content.removeChild(content.firstChild);
    const p = document.createElement('p');
    p.style.padding = '24px';
    p.textContent = `Unknown route: ${hash}`;
    content.appendChild(p);
  }
}

window.addEventListener('hashchange', render);

// ── Bootstrap ─────────────────────────────────────────────────────────────────
import './auth.js';
import './map.js';
import './history.js';
import './places.js';
import './circle.js';
import './profile.js';

// Trigger initial render after imports
render();
