import { api, getActiveCircleId, registerRoute } from './app.js';

registerRoute('history', async (content) => {
  // Outer wrapper fills #content absolutely
  const wrapper = document.createElement('div');
  wrapper.className = 'history-wrapper';
  content.appendChild(wrapper);

  // Controls bar
  const controls = document.createElement('div');
  controls.className = 'history-controls';

  const select = document.createElement('select');
  select.id = 'history-member-select';

  const defaultOpt = document.createElement('option');
  defaultOpt.value = '';
  defaultOpt.textContent = 'Select member…';
  select.appendChild(defaultOpt);

  const today = new Date().toISOString().slice(0, 10);
  const dateInput = document.createElement('input');
  dateInput.type = 'date';
  dateInput.id = 'history-date';
  dateInput.value = today;

  const showBtn = document.createElement('button');
  showBtn.className = 'btn btn-primary btn-small';
  showBtn.style.width = 'auto';
  showBtn.textContent = 'Show Path';

  controls.appendChild(select);
  controls.appendChild(dateInput);
  controls.appendChild(showBtn);
  wrapper.appendChild(controls);

  // Map area
  const mapArea = document.createElement('div');
  mapArea.className = 'history-map';
  const mapEl = document.createElement('div');
  mapEl.id = 'history-leaflet-map';
  mapArea.appendChild(mapEl);
  wrapper.appendChild(mapArea);

  // Init Leaflet
  // eslint-disable-next-line no-undef
  L.Icon.Default.imagePath = '/lib/leaflet/images/';
  // eslint-disable-next-line no-undef
  const map = L.map('history-leaflet-map').setView([0, 0], 2);
  // eslint-disable-next-line no-undef
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '© OpenStreetMap contributors',
    maxZoom: 19,
  }).addTo(map);

  let polyline = null;

  // Load members into select
  const circleId = getActiveCircleId();
  if (circleId) {
    try {
      const members = await api('GET', `/circles/${circleId}/members`);
      if (Array.isArray(members)) {
        members.forEach(m => {
          const opt = document.createElement('option');
          opt.value = m.user_id || m.id;
          opt.textContent = m.display_name || m.name || 'Unknown';
          select.appendChild(opt);
        });
      }
    } catch { /* ignore */ }
  }

  // Show path on button click
  showBtn.addEventListener('click', async () => {
    const userId = select.value;
    const date   = dateInput.value;
    if (!userId || !date) return;

    const from = `${date}T00:00:00Z`;
    const to   = `${date}T23:59:59Z`;

    // Remove existing polyline
    if (polyline) {
      map.removeLayer(polyline);
      polyline = null;
    }

    try {
      const locs = await api('GET', `/locations/history?user_id=${userId}&from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`);
      if (!Array.isArray(locs) || locs.length === 0) return;

      const latlngs = locs.map(l => [l.lat, l.lng]);
      // eslint-disable-next-line no-undef
      polyline = L.polyline(latlngs, { color: '#1B73E8', weight: 4 }).addTo(map);
      map.fitBounds(polyline.getBounds(), { padding: [40, 40] });
    } catch { /* ignore */ }
  });

  // Cleanup on route change
  window.addEventListener('hashchange', function onHashChange() {
    window.removeEventListener('hashchange', onHashChange);
    map.remove();
  }, { once: true });
});
