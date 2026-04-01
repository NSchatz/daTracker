import { getUser, getServerUrl, setServerUrl, clearAuth, navigate, registerRoute } from './app.js';

registerRoute('profile', (content) => {
  const panel = document.createElement('div');
  panel.className = 'panel';

  const h2 = document.createElement('h2');
  h2.textContent = 'Profile';
  panel.appendChild(h2);

  const user = getUser();

  // Display name field
  const nameField = document.createElement('div');
  nameField.className = 'profile-field';
  const nameLabel = document.createElement('div');
  nameLabel.className = 'profile-field-label';
  nameLabel.textContent = 'Display Name';
  const nameValue = document.createElement('div');
  nameValue.className = 'profile-field-value';
  nameValue.textContent = user ? (user.display_name || user.name || '—') : '—';
  nameField.appendChild(nameLabel);
  nameField.appendChild(nameValue);
  panel.appendChild(nameField);

  // Email field
  const emailField = document.createElement('div');
  emailField.className = 'profile-field';
  const emailLabel = document.createElement('div');
  emailLabel.className = 'profile-field-label';
  emailLabel.textContent = 'Email';
  const emailValue = document.createElement('div');
  emailValue.className = 'profile-field-value';
  emailValue.textContent = user ? (user.email || '—') : '—';
  emailField.appendChild(emailLabel);
  emailField.appendChild(emailValue);
  panel.appendChild(emailField);

  // Server URL (editable)
  const serverGroup = document.createElement('div');
  serverGroup.className = 'form-group';
  serverGroup.style.marginTop = '20px';
  const serverLabel = document.createElement('label');
  serverLabel.textContent = 'Server URL';
  serverLabel.htmlFor = 'profile-server-url';
  const serverInput = document.createElement('input');
  serverInput.type = 'url';
  serverInput.id = 'profile-server-url';
  serverInput.value = getServerUrl();
  serverGroup.appendChild(serverLabel);
  serverGroup.appendChild(serverInput);
  panel.appendChild(serverGroup);

  const saveBtn = document.createElement('button');
  saveBtn.className = 'btn btn-primary';
  saveBtn.style.width = 'auto';
  saveBtn.textContent = 'Save';

  const feedback = document.createElement('p');
  feedback.className = 'feedback-msg';
  feedback.textContent = 'Saved!';

  saveBtn.addEventListener('click', () => {
    const url = serverInput.value.trim();
    if (url) {
      setServerUrl(url);
      feedback.classList.add('visible');
      setTimeout(() => feedback.classList.remove('visible'), 2000);
    }
  });

  panel.appendChild(saveBtn);
  panel.appendChild(feedback);

  // Logout
  const divider = document.createElement('hr');
  divider.style.margin = '24px 0';
  divider.style.borderColor = '#eee';
  panel.appendChild(divider);

  const logoutBtn = document.createElement('button');
  logoutBtn.className = 'btn btn-outline';
  logoutBtn.style.color = '#dc2626';
  logoutBtn.style.borderColor = '#dc2626';
  logoutBtn.textContent = 'Logout';
  logoutBtn.addEventListener('click', () => {
    clearAuth();
    navigate('login');
  });
  panel.appendChild(logoutBtn);

  content.appendChild(panel);
});
