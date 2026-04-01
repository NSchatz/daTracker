import { api, getActiveCircleId, registerRoute } from './app.js';

registerRoute('circle', async (content) => {
  const panel = document.createElement('div');
  panel.className = 'panel';

  const h2 = document.createElement('h2');
  h2.textContent = 'Circle';
  panel.appendChild(h2);

  content.appendChild(panel);

  const circleId = getActiveCircleId();
  if (!circleId) {
    const msg = document.createElement('p');
    msg.style.color = '#888';
    msg.textContent = 'No active circle found.';
    panel.appendChild(msg);
    return;
  }

  // Load circles and find active
  let circle = null;
  try {
    const circles = await api('GET', '/circles');
    if (Array.isArray(circles)) {
      circle = circles.find(c => String(c.id) === String(circleId)) || circles[0] || null;
    }
  } catch { /* ignore */ }

  if (!circle) {
    const msg = document.createElement('p');
    msg.style.color = '#888';
    msg.textContent = 'Could not load circle info.';
    panel.appendChild(msg);
    return;
  }

  // Circle name
  const nameEl = document.createElement('div');
  nameEl.className = 'profile-field';
  const nameLabel = document.createElement('div');
  nameLabel.className = 'profile-field-label';
  nameLabel.textContent = 'Circle Name';
  const nameValue = document.createElement('div');
  nameValue.className = 'profile-field-value';
  nameValue.textContent = circle.name;
  nameEl.appendChild(nameLabel);
  nameEl.appendChild(nameValue);
  panel.appendChild(nameEl);

  // Invite code
  const sectionLabel = document.createElement('div');
  sectionLabel.className = 'section-label';
  sectionLabel.textContent = 'Invite Code';
  panel.appendChild(sectionLabel);

  const inviteBox = document.createElement('div');
  inviteBox.className = 'invite-box';

  const codeEl = document.createElement('code');
  codeEl.textContent = circle.invite_code || 'N/A';

  const copyBtn = document.createElement('button');
  copyBtn.className = 'btn btn-outline btn-small';
  copyBtn.textContent = 'Copy';
  copyBtn.addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(circle.invite_code || '');
      copyBtn.textContent = 'Copied!';
      setTimeout(() => { copyBtn.textContent = 'Copy'; }, 2000);
    } catch {
      copyBtn.textContent = 'Failed';
      setTimeout(() => { copyBtn.textContent = 'Copy'; }, 2000);
    }
  });

  inviteBox.appendChild(codeEl);
  inviteBox.appendChild(copyBtn);
  panel.appendChild(inviteBox);

  // Members list
  const membersLabel = document.createElement('div');
  membersLabel.className = 'section-label';
  membersLabel.textContent = 'Members';
  panel.appendChild(membersLabel);

  try {
    const members = await api('GET', `/circles/${circleId}/members`);
    if (Array.isArray(members) && members.length > 0) {
      members.forEach(m => {
        const item = document.createElement('div');
        item.className = 'list-item';

        const main = document.createElement('div');
        main.className = 'list-item-main';

        const nameDiv = document.createElement('div');
        nameDiv.className = 'list-item-title';
        nameDiv.textContent = m.display_name || m.name || 'Unknown';

        const emailDiv = document.createElement('div');
        emailDiv.className = 'list-item-sub';
        emailDiv.textContent = m.email || '';

        main.appendChild(nameDiv);
        main.appendChild(emailDiv);
        item.appendChild(main);

        if (m.role) {
          const badge = document.createElement('span');
          badge.className = 'badge';
          badge.textContent = m.role;
          item.appendChild(badge);
        }

        panel.appendChild(item);
      });
    } else {
      const empty = document.createElement('p');
      empty.style.color = '#888';
      empty.textContent = 'No members found.';
      panel.appendChild(empty);
    }
  } catch { /* ignore */ }
});
