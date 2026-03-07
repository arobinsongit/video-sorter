import { state } from './state.js';
import { clearChildren } from './utils.js';

let cloudModal = null;

export function setupCloud() {
  const btn = document.getElementById('btnCloud');
  if (btn) btn.addEventListener('click', openCloudModal);
}

async function fetchProviders() {
  try {
    const resp = await fetch('/api/cloud/providers');
    return await resp.json();
  } catch (e) {
    return [];
  }
}

async function connectProvider(id) {
  const resp = await fetch('/api/cloud/connect', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ provider: id })
  });
  const data = await resp.json();
  if (data.error) {
    alert(data.error);
    return;
  }
  if (data.authURL) {
    window.open(data.authURL, '_blank', 'width=600,height=700');
    // Poll for connection status
    pollConnection(id);
  }
}

async function disconnectProvider(id) {
  await fetch('/api/cloud/disconnect', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ provider: id })
  });
  renderCloudModal();
}

function pollConnection(id) {
  let attempts = 0;
  const interval = setInterval(async () => {
    attempts++;
    if (attempts > 60) { clearInterval(interval); return; } // 2 min timeout
    const providers = await fetchProviders();
    const provider = providers.find(p => p.id === id);
    if (provider && provider.connected) {
      clearInterval(interval);
      renderCloudModal();
    }
  }, 2000);
}

async function browseDrive(path) {
  try {
    const resp = await fetch('/api/cloud/browse?provider=gdrive&path=' + encodeURIComponent(path || ''));
    return await resp.json();
  } catch (e) {
    return [];
  }
}

function openCloudModal() {
  cloudModal = document.getElementById('cloudModal');
  if (!cloudModal) return;
  cloudModal.classList.remove('hidden');
  cloudModal.classList.add('flex');
  renderCloudModal();
}

function closeCloudModal() {
  if (!cloudModal) return;
  cloudModal.classList.add('hidden');
  cloudModal.classList.remove('flex');
}

async function renderCloudModal() {
  const content = document.getElementById('cloudModalContent');
  if (!content) return;
  clearChildren(content);

  const providers = await fetchProviders();

  providers.forEach(p => {
    const card = document.createElement('div');
    card.className = 'flex items-center justify-between p-3 border border-neutral-200 dark:border-neutral-700 rounded-lg';

    const left = document.createElement('div');
    const name = document.createElement('div');
    name.className = 'text-sm font-medium text-neutral-900 dark:text-neutral-100';
    name.textContent = p.name;
    left.appendChild(name);

    const status = document.createElement('div');
    status.className = 'text-xs ' + (p.connected ? 'text-green-600 dark:text-green-400' : 'text-neutral-400');
    status.textContent = p.connected ? 'Connected' : (p.hasCreds ? 'Not connected' : 'Credentials not configured');
    left.appendChild(status);

    card.appendChild(left);

    if (p.connected) {
      const disconnectBtn = document.createElement('button');
      disconnectBtn.className = 'px-3 py-1.5 text-xs rounded-md border border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 cursor-pointer';
      disconnectBtn.textContent = 'Disconnect';
      disconnectBtn.addEventListener('click', () => disconnectProvider(p.id));
      card.appendChild(disconnectBtn);
    } else if (p.hasCreds) {
      const connectBtn = document.createElement('button');
      connectBtn.className = 'px-3 py-1.5 text-xs rounded-md bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 hover:bg-neutral-700 dark:hover:bg-neutral-300 cursor-pointer';
      connectBtn.textContent = 'Connect';
      connectBtn.addEventListener('click', () => connectProvider(p.id));
      card.appendChild(connectBtn);
    }

    content.appendChild(card);
  });

  // Browse section for connected Google Drive
  const gdriveProvider = providers.find(p => p.id === 'gdrive' && p.connected);
  if (gdriveProvider) {
    const browseSection = document.createElement('div');
    browseSection.className = 'mt-4 pt-4 border-t border-neutral-200 dark:border-neutral-700';

    const browseTitle = document.createElement('div');
    browseTitle.className = 'text-xs font-medium uppercase tracking-wide text-neutral-500 mb-2';
    browseTitle.textContent = 'Browse Google Drive';
    browseSection.appendChild(browseTitle);

    const pathDisplay = document.createElement('div');
    pathDisplay.className = 'text-xs text-neutral-500 mb-2 font-mono';
    pathDisplay.textContent = 'gdrive://';
    browseSection.appendChild(pathDisplay);

    const folderList = document.createElement('div');
    folderList.className = 'flex flex-col gap-1 max-h-48 overflow-y-auto';
    browseSection.appendChild(folderList);

    let currentPath = '';

    async function renderFolders(path) {
      currentPath = path;
      pathDisplay.textContent = 'gdrive://' + (path || '');
      clearChildren(folderList);

      if (path) {
        const upBtn = document.createElement('button');
        upBtn.className = 'text-left px-2 py-1 text-xs text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800 rounded cursor-pointer';
        upBtn.textContent = '.. (up)';
        upBtn.addEventListener('click', () => {
          const parts = path.split('/').filter(Boolean);
          parts.pop();
          renderFolders(parts.join('/'));
        });
        folderList.appendChild(upBtn);
      }

      const folders = await browseDrive(path);
      if (folders.error) return;

      folders.forEach(f => {
        const btn = document.createElement('button');
        btn.className = 'text-left px-2 py-1 text-xs text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 rounded cursor-pointer';
        btn.textContent = '\uD83D\uDCC1 ' + f.name;
        btn.addEventListener('click', () => renderFolders(f.path));
        folderList.appendChild(btn);
      });

      if (Array.isArray(folders) && folders.length === 0 && !path) {
        const empty = document.createElement('div');
        empty.className = 'text-xs text-neutral-400 px-2 py-1';
        empty.textContent = 'No folders found';
        folderList.appendChild(empty);
      }
    }

    // Use folder button
    const useBtn = document.createElement('button');
    useBtn.className = 'mt-2 w-full py-1.5 text-xs font-medium rounded-md bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 hover:bg-neutral-700 dark:hover:bg-neutral-300 cursor-pointer';
    useBtn.textContent = 'Use This Folder';
    useBtn.addEventListener('click', () => {
      const dirInput = document.getElementById('dirInput');
      if (dirInput) {
        dirInput.value = 'gdrive://' + currentPath;
      }
      closeCloudModal();
    });
    browseSection.appendChild(useBtn);

    content.appendChild(browseSection);
    renderFolders('');
  }

  // Setup help text
  const gdriveNotReady = providers.find(p => p.id === 'gdrive' && !p.hasCreds);
  if (gdriveNotReady) {
    const help = document.createElement('div');
    help.className = 'mt-3 p-3 text-xs text-neutral-500 bg-neutral-50 dark:bg-neutral-900 rounded-lg';
    help.textContent = 'To use Google Drive: place your OAuth client credentials JSON file at ~/.media-sorter/gdrive-credentials.json';
    content.appendChild(help);
  }
}

export { closeCloudModal };
