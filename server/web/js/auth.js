import { api, saveAuth, getServerUrl, setServerUrl, navigate, registerRoute } from './app.js';

// ── Helpers ───────────────────────────────────────────────────────────────────
function showError(el, msg) {
  el.textContent = msg;
  el.classList.add('visible');
}

function hideError(el) {
  el.textContent = '';
  el.classList.remove('visible');
}

// ── Login route ───────────────────────────────────────────────────────────────
registerRoute('login', (content) => {
  const container = document.createElement('div');
  container.className = 'auth-container';

  // Static structure - no user data interpolated
  container.innerHTML = `
    <h2>Sign In</h2>
    <div class="form-group">
      <label for="login-server">Server URL</label>
      <input id="login-server" type="url" placeholder="https://tracker.example.com" />
    </div>
    <div class="form-group">
      <label for="login-email">Email</label>
      <input id="login-email" type="email" placeholder="you@example.com" autocomplete="username" />
    </div>
    <div class="form-group">
      <label for="login-password">Password</label>
      <input id="login-password" type="password" placeholder="••••••••" autocomplete="current-password" />
    </div>
    <button class="btn btn-primary" id="login-btn">Login</button>
    <p class="error-msg" id="login-error"></p>
    <div style="text-align:center;margin-top:16px">
      <button class="btn-link" id="goto-register">Create Account</button>
    </div>
  `;

  content.appendChild(container);

  const serverInput = document.getElementById('login-server');
  const emailInput  = document.getElementById('login-email');
  const passInput   = document.getElementById('login-password');
  const loginBtn    = document.getElementById('login-btn');
  const errorEl     = document.getElementById('login-error');

  // Pre-fill server URL
  serverInput.value = getServerUrl();

  async function doLogin() {
    hideError(errorEl);
    const serverUrl = serverInput.value.trim();
    const email     = emailInput.value.trim();
    const password  = passInput.value;

    if (!serverUrl) { showError(errorEl, 'Server URL is required'); return; }
    if (!email)     { showError(errorEl, 'Email is required'); return; }
    if (!password)  { showError(errorEl, 'Password is required'); return; }

    setServerUrl(serverUrl);
    loginBtn.disabled = true;
    loginBtn.textContent = 'Signing in…';

    try {
      const data = await api('POST', '/auth/login', { email, password });
      saveAuth(data.token, data.user);
      navigate('map');
    } catch (err) {
      showError(errorEl, err.message || 'Login failed');
      loginBtn.disabled = false;
      loginBtn.textContent = 'Login';
    }
  }

  loginBtn.addEventListener('click', doLogin);
  passInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') doLogin(); });

  document.getElementById('goto-register').addEventListener('click', () => navigate('register'));
});

// ── Register route ────────────────────────────────────────────────────────────
registerRoute('register', (content) => {
  const container = document.createElement('div');
  container.className = 'auth-container';

  // Static structure - no user data interpolated
  container.innerHTML = `
    <h2>Create Account</h2>
    <div class="form-group">
      <label for="reg-name">Display Name</label>
      <input id="reg-name" type="text" placeholder="Your name" autocomplete="name" />
    </div>
    <div class="form-group">
      <label for="reg-email">Email</label>
      <input id="reg-email" type="email" placeholder="you@example.com" autocomplete="username" />
    </div>
    <div class="form-group">
      <label for="reg-password">Password</label>
      <input id="reg-password" type="password" placeholder="••••••••" autocomplete="new-password" />
    </div>
    <div class="form-group">
      <label for="reg-invite">Invite Code</label>
      <input id="reg-invite" type="text" placeholder="Enter invite code" />
    </div>
    <button class="btn btn-primary" id="register-btn">Register</button>
    <p class="error-msg" id="register-error"></p>
    <div style="text-align:center;margin-top:16px">
      <button class="btn-link" id="goto-login">Back to Login</button>
    </div>
  `;

  content.appendChild(container);

  const nameInput   = document.getElementById('reg-name');
  const emailInput  = document.getElementById('reg-email');
  const passInput   = document.getElementById('reg-password');
  const inviteInput = document.getElementById('reg-invite');
  const regBtn      = document.getElementById('register-btn');
  const errorEl     = document.getElementById('register-error');

  async function doRegister() {
    hideError(errorEl);
    const display_name  = nameInput.value.trim();
    const email         = emailInput.value.trim();
    const password      = passInput.value;
    const invite_code   = inviteInput.value.trim();

    if (!display_name) { showError(errorEl, 'Display name is required'); return; }
    if (!email)        { showError(errorEl, 'Email is required'); return; }
    if (!password)     { showError(errorEl, 'Password is required'); return; }
    if (!invite_code)  { showError(errorEl, 'Invite code is required'); return; }

    regBtn.disabled = true;
    regBtn.textContent = 'Creating account…';

    try {
      const data = await api('POST', '/auth/register', { email, display_name, password, invite_code });
      saveAuth(data.token, data.user);
      navigate('map');
    } catch (err) {
      showError(errorEl, err.message || 'Registration failed');
      regBtn.disabled = false;
      regBtn.textContent = 'Register';
    }
  }

  regBtn.addEventListener('click', doRegister);

  document.getElementById('goto-login').addEventListener('click', () => navigate('login'));
});
