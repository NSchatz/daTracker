import { api, getActiveCircleId, registerRoute } from './app.js';

registerRoute('places', async (content) => {
  const panel = document.createElement('div');
  panel.className = 'panel';

  // Header
  const header = document.createElement('div');
  header.className = 'panel-header';

  const h2 = document.createElement('h2');
  h2.textContent = 'Places';

  const addBtn = document.createElement('button');
  addBtn.className = 'btn btn-primary btn-small';
  addBtn.style.width = 'auto';
  addBtn.textContent = 'Add Place';

  header.appendChild(h2);
  header.appendChild(addBtn);
  panel.appendChild(header);

  const list = document.createElement('div');
  list.id = 'places-list';
  panel.appendChild(list);

  content.appendChild(panel);

  const circleId = getActiveCircleId();

  async function loadGeofences() {
    // Clear list
    while (list.firstChild) list.removeChild(list.firstChild);

    if (!circleId) return;
    try {
      const fences = await api('GET', `/geofences?circle_id=${circleId}`);
      if (!Array.isArray(fences) || fences.length === 0) {
        const empty = document.createElement('p');
        empty.style.color = '#888';
        empty.textContent = 'No places yet. Add one!';
        list.appendChild(empty);
        return;
      }
      fences.forEach(gf => {
        const item = document.createElement('div');
        item.className = 'list-item';

        const main = document.createElement('div');
        main.className = 'list-item-main';

        const title = document.createElement('div');
        title.className = 'list-item-title';
        title.textContent = gf.name;

        const sub = document.createElement('div');
        sub.className = 'list-item-sub';
        sub.textContent = `Radius: ${gf.radius_meters}m`;

        main.appendChild(title);
        main.appendChild(sub);

        const delBtn = document.createElement('button');
        delBtn.className = 'btn-icon';
        delBtn.title = 'Delete';
        delBtn.textContent = '✕';
        delBtn.addEventListener('click', async () => {
          if (!window.confirm(`Delete "${gf.name}"?`)) return;
          try {
            await api('DELETE', `/geofences/${gf.id}`);
            loadGeofences();
          } catch (err) {
            alert(err.message || 'Delete failed');
          }
        });

        item.appendChild(main);
        item.appendChild(delBtn);
        list.appendChild(item);
      });
    } catch { /* ignore */ }
  }

  await loadGeofences();

  // Add Place modal
  addBtn.addEventListener('click', () => {
    openAddModal(circleId, loadGeofences);
  });
});

function openAddModal(circleId, onCreated) {
  // Overlay
  const overlay = document.createElement('div');
  overlay.className = 'modal-overlay';

  const modal = document.createElement('div');
  modal.className = 'modal';

  const h3 = document.createElement('h3');
  h3.textContent = 'Add Place';
  modal.appendChild(h3);

  // Name field
  const nameGroup = document.createElement('div');
  nameGroup.className = 'form-group';
  const nameLabel = document.createElement('label');
  nameLabel.textContent = 'Name';
  nameLabel.htmlFor = 'place-name';
  const nameInput = document.createElement('input');
  nameInput.type = 'text';
  nameInput.id = 'place-name';
  nameInput.placeholder = 'Home, Work…';
  nameGroup.appendChild(nameLabel);
  nameGroup.appendChild(nameInput);
  modal.appendChild(nameGroup);

  // Radius field
  const radGroup = document.createElement('div');
  radGroup.className = 'form-group';
  const radLabel = document.createElement('label');
  radLabel.textContent = 'Radius (meters)';
  radLabel.htmlFor = 'place-radius';
  const radInput = document.createElement('input');
  radInput.type = 'number';
  radInput.id = 'place-radius';
  radInput.value = '100';
  radInput.min = '10';
  radGroup.appendChild(radLabel);
  radGroup.appendChild(radInput);
  modal.appendChild(radGroup);

  // Map hint
  const hint = document.createElement('p');
  hint.style.fontSize = '0.82rem';
  hint.style.color = '#888';
  hint.style.marginBottom = '4px';
  hint.textContent = 'Click the map to place the pin';
  modal.appendChild(hint);

  // Leaflet mini-map
  const mapDiv = document.createElement('div');
  mapDiv.className = 'modal-map';
  const mapEl = document.createElement('div');
  mapEl.id = 'modal-leaflet-map';
  mapDiv.appendChild(mapEl);
  modal.appendChild(mapDiv);

  // Actions
  const actions = document.createElement('div');
  actions.className = 'modal-actions';

  const cancelBtn = document.createElement('button');
  cancelBtn.className = 'btn btn-outline btn-small';
  cancelBtn.textContent = 'Cancel';

  const createBtn = document.createElement('button');
  createBtn.className = 'btn btn-primary btn-small';
  createBtn.style.width = 'auto';
  createBtn.textContent = 'Create';

  actions.appendChild(cancelBtn);
  actions.appendChild(createBtn);
  modal.appendChild(actions);

  overlay.appendChild(modal);
  document.body.appendChild(overlay);

  // Init mini-map after DOM insertion
  // eslint-disable-next-line no-undef
  L.Icon.Default.imagePath = '/lib/leaflet/images/';
  // eslint-disable-next-line no-undef
  const miniMap = L.map('modal-leaflet-map').setView([0, 0], 2);
  // eslint-disable-next-line no-undef
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '© OpenStreetMap contributors',
    maxZoom: 19,
  }).addTo(miniMap);

  let pinMarker = null;
  let pickedLat = null;
  let pickedLng = null;

  miniMap.on('click', (e) => {
    pickedLat = e.latlng.lat;
    pickedLng = e.latlng.lng;
    if (pinMarker) {
      pinMarker.setLatLng(e.latlng);
    } else {
      // eslint-disable-next-line no-undef
      pinMarker = L.marker(e.latlng).addTo(miniMap);
    }
  });

  function closeModal() {
    miniMap.remove();
    document.body.removeChild(overlay);
  }

  cancelBtn.addEventListener('click', closeModal);

  createBtn.addEventListener('click', async () => {
    const name   = nameInput.value.trim();
    const radius = parseInt(radInput.value, 10);

    if (!name)          { alert('Name is required'); return; }
    if (!pickedLat)     { alert('Click the map to set a location'); return; }
    if (!radius || radius < 10) { alert('Radius must be at least 10m'); return; }

    createBtn.disabled = true;
    createBtn.textContent = 'Creating…';

    try {
      await api('POST', '/geofences', {
        circle_id:      circleId,
        name,
        lat:            pickedLat,
        lng:            pickedLng,
        radius_meters:  radius,
      });
      closeModal();
      onCreated();
    } catch (err) {
      alert(err.message || 'Create failed');
      createBtn.disabled = false;
      createBtn.textContent = 'Create';
    }
  });
}
