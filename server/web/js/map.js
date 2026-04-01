import { api, getToken, getActiveCircleId, getServerUrl, navigate, registerRoute } from './app.js';

function formatTime(iso) {
  if (!iso) return 'unknown';
  return new Date(iso).toLocaleTimeString();
}

function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = String(str);
  return div.innerHTML;
}

registerRoute('map', async (content) => {
  // Build map container
  const wrapper = document.createElement('div');
  wrapper.className = 'map-container';
  const mapEl = document.createElement('div');
  mapEl.id = 'leaflet-map';
  wrapper.appendChild(mapEl);
  content.appendChild(wrapper);

  // Set Leaflet image path
  // eslint-disable-next-line no-undef
  L.Icon.Default.imagePath = '/lib/leaflet/images/';

  // Create map
  // eslint-disable-next-line no-undef
  const map = L.map('leaflet-map').setView([0, 0], 2);
  // eslint-disable-next-line no-undef
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '© OpenStreetMap contributors',
    maxZoom: 19,
  }).addTo(map);

  const markers = {}; // userId → L.marker
  let ws = null;

  const circleId = getActiveCircleId();
  if (!circleId) {
    navigate('circle');
    return;
  }

  // Load member name lookup
  const memberNames = {};
  try {
    const members = await api('GET', `/circles/${circleId}/members`);
    if (Array.isArray(members)) {
      members.forEach(m => { memberNames[m.user_id || m.id] = m.display_name || m.name || 'Unknown'; });
    }
  } catch { /* ignore */ }

  // Load latest positions
  try {
    const locations = await api('GET', `/locations/latest?circle_id=${circleId}`);
    if (Array.isArray(locations) && locations.length > 0) {
      const bounds = [];
      locations.forEach(loc => {
        const name = escapeHtml(memberNames[loc.user_id] || 'Unknown');
        const time = escapeHtml(formatTime(loc.timestamp));
        const battery = loc.battery_level != null ? escapeHtml(String(loc.battery_level) + '%') : 'N/A';
        // eslint-disable-next-line no-undef
        const marker = L.marker([loc.lat, loc.lng])
          .addTo(map)
          .bindPopup(`<strong>${name}</strong><br>Time: ${time}<br>Battery: ${battery}`);
        markers[loc.user_id] = marker;
        bounds.push([loc.lat, loc.lng]);
      });
      if (bounds.length > 0) {
        // eslint-disable-next-line no-undef
        map.fitBounds(bounds, { padding: [40, 40] });
      }
    }
  } catch { /* ignore */ }

  // Load geofences
  try {
    const geofences = await api('GET', `/geofences?circle_id=${circleId}`);
    if (Array.isArray(geofences)) {
      geofences.forEach(gf => {
        // eslint-disable-next-line no-undef
        L.circle([gf.lat, gf.lng], {
          radius: gf.radius_meters,
          color: '#1B73E8',
          fillColor: '#1B73E8',
          fillOpacity: 0.15,
          weight: 2,
        }).addTo(map).bindPopup(escapeHtml(gf.name));
      });
    }
  } catch { /* ignore */ }

  // WebSocket live updates
  function connectWs() {
    if (!document.getElementById('leaflet-map')) return; // map was destroyed

    const token = getToken();
    const base = getServerUrl().replace(/^http/, 'ws');
    ws = new WebSocket(`${base}/ws?circle_id=${circleId}&token=${token}`);

    ws.addEventListener('message', (evt) => {
      try {
        const loc = JSON.parse(evt.data);
        const name = escapeHtml(memberNames[loc.user_id] || 'Unknown');
        const time = escapeHtml(formatTime(loc.timestamp));
        const battery = loc.battery_level != null ? escapeHtml(String(loc.battery_level) + '%') : 'N/A';
        const popup = `<strong>${name}</strong><br>Time: ${time}<br>Battery: ${battery}`;
        if (markers[loc.user_id]) {
          markers[loc.user_id].setLatLng([loc.lat, loc.lng]).setPopupContent(popup);
        } else {
          // eslint-disable-next-line no-undef
          markers[loc.user_id] = L.marker([loc.lat, loc.lng])
            .addTo(map)
            .bindPopup(popup);
        }
      } catch { /* ignore malformed */ }
    });

    ws.addEventListener('close', () => {
      if (!document.getElementById('leaflet-map')) return;
      setTimeout(connectWs, 5000);
    });
  }

  connectWs();

  // Cleanup when route changes
  const origHashChange = window._mapCleanup;
  window._mapCleanup = () => {
    if (ws) { ws.close(); ws = null; }
    map.remove();
    if (origHashChange) origHashChange();
  };

  window.addEventListener('hashchange', function onHashChange() {
    window.removeEventListener('hashchange', onHashChange);
    if (ws) { ws.close(); ws = null; }
    map.remove();
  }, { once: true });
});
