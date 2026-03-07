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

async function saveCredentials(provider, credentials) {
  const resp = await fetch('/api/cloud/credentials', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ provider, credentials })
  });
  return await resp.json();
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
    pollConnection(id);
  } else if (data.status === 'connected') {
    renderCloudModal();
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
    if (attempts > 60) { clearInterval(interval); return; }
    const providers = await fetchProviders();
    const provider = providers.find(p => p.id === id);
    if (provider && provider.connected) {
      clearInterval(interval);
      renderCloudModal();
    }
  }, 2000);
}

async function browseProvider(providerId, path) {
  try {
    const resp = await fetch('/api/cloud/browse?provider=' + encodeURIComponent(providerId) + '&path=' + encodeURIComponent(path || ''));
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

// Credential form field definitions per provider
const credentialFields = {
  s3: {
    type: 'fields',
    fields: [
      { key: 'accessKeyId', label: 'Access Key ID', required: true },
      { key: 'secretAccessKey', label: 'Secret Access Key', required: true, secret: true },
      { key: 'region', label: 'Region', required: true, placeholder: 'us-east-1' },
      { key: 'endpoint', label: 'Custom Endpoint (optional)', required: false, placeholder: 'https://s3.example.com' }
    ],
    help: 'Create an IAM user with S3 access in AWS Console'
  },
  dropbox: {
    type: 'fields',
    fields: [
      { key: 'clientId', label: 'App Key', required: true },
      { key: 'clientSecret', label: 'App Secret', required: true, secret: true }
    ],
    help: 'Create an app at dropbox.com/developers/apps'
  },
  onedrive: {
    type: 'fields',
    fields: [
      { key: 'clientId', label: 'Application (Client) ID', required: true },
      { key: 'clientSecret', label: 'Client Secret', required: true, secret: true }
    ],
    help: 'Register an app at portal.azure.com > App registrations'
  }
};

function createCredentialForm(providerId, onSaved) {
  const config = credentialFields[providerId];
  if (!config) return null;

  const form = document.createElement('div');
  form.className = 'mt-2 p-3 bg-neutral-50 dark:bg-neutral-900 rounded-lg space-y-2';

  // Help text
  const help = document.createElement('div');
  help.className = 'text-xs text-neutral-400 mb-2';
  help.textContent = config.help;
  form.appendChild(help);

  if (config.type === 'json') {
    // JSON paste area (for Google Drive)
    const textarea = document.createElement('textarea');
    textarea.className = 'w-full h-24 text-xs font-mono p-2 rounded border border-neutral-300 dark:border-neutral-600 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-neutral-100 resize-none';
    textarea.placeholder = '{"installed":{"client_id":"...","client_secret":"...",...}}';
    form.appendChild(textarea);

    const btnRow = document.createElement('div');
    btnRow.className = 'flex gap-2';

    const saveBtn = document.createElement('button');
    saveBtn.className = 'px-3 py-1.5 text-xs rounded-md bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 hover:bg-neutral-700 dark:hover:bg-neutral-300 cursor-pointer';
    saveBtn.textContent = 'Save Credentials';
    saveBtn.addEventListener('click', async () => {
      const val = textarea.value.trim();
      if (!val) { alert('Please paste the credentials JSON'); return; }
      let parsed;
      try { parsed = JSON.parse(val); } catch (e) { alert('Invalid JSON: ' + e.message); return; }
      saveBtn.disabled = true;
      saveBtn.textContent = 'Saving...';
      const result = await saveCredentials(providerId, parsed);
      if (result.error) {
        alert(result.error);
        saveBtn.disabled = false;
        saveBtn.textContent = 'Save Credentials';
      } else {
        onSaved();
      }
    });
    btnRow.appendChild(saveBtn);
    form.appendChild(btnRow);
  } else {
    // Individual fields
    const inputs = {};
    config.fields.forEach(f => {
      const label = document.createElement('label');
      label.className = 'block text-xs text-neutral-600 dark:text-neutral-400';
      label.textContent = f.label + (f.required ? ' *' : '');

      const input = document.createElement('input');
      input.type = f.secret ? 'password' : 'text';
      input.className = 'mt-0.5 w-full text-xs p-1.5 rounded border border-neutral-300 dark:border-neutral-600 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-neutral-100';
      if (f.placeholder) input.placeholder = f.placeholder;

      label.appendChild(input);
      form.appendChild(label);
      inputs[f.key] = input;
    });

    const saveBtn = document.createElement('button');
    saveBtn.className = 'mt-1 px-3 py-1.5 text-xs rounded-md bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 hover:bg-neutral-700 dark:hover:bg-neutral-300 cursor-pointer';
    saveBtn.textContent = 'Save & Connect';
    saveBtn.addEventListener('click', async () => {
      const creds = {};
      for (const f of config.fields) {
        const val = inputs[f.key].value.trim();
        if (f.required && !val) { alert(f.label + ' is required'); return; }
        if (val) creds[f.key] = val;
      }
      saveBtn.disabled = true;
      saveBtn.textContent = 'Saving...';
      const result = await saveCredentials(providerId, creds);
      if (result.error) {
        alert(result.error);
        saveBtn.disabled = false;
        saveBtn.textContent = 'Save & Connect';
      } else {
        onSaved();
      }
    });
    form.appendChild(saveBtn);
  }

  return form;
}

async function renderCloudModal() {
  const content = document.getElementById('cloudModalContent');
  if (!content) return;
  clearChildren(content);

  const providers = await fetchProviders();

  providers.forEach(p => {
    const card = document.createElement('div');
    card.className = 'p-3 border border-neutral-200 dark:border-neutral-700 rounded-lg';

    const header = document.createElement('div');
    header.className = 'flex items-center justify-between';

    const left = document.createElement('div');
    const name = document.createElement('div');
    name.className = 'text-sm font-medium text-neutral-900 dark:text-neutral-100';
    name.textContent = p.name;
    left.appendChild(name);

    const status = document.createElement('div');
    status.className = 'text-xs ' + (p.connected ? 'text-green-600 dark:text-green-400' : (p.hasCreds ? 'text-amber-500' : 'text-neutral-400'));
    status.textContent = p.connected ? 'Connected' : (p.hasCreds ? 'Ready to connect' : 'Not configured');
    left.appendChild(status);
    header.appendChild(left);

    const btnGroup = document.createElement('div');
    btnGroup.className = 'flex gap-2';

    if (p.connected) {
      const disconnectBtn = document.createElement('button');
      disconnectBtn.className = 'px-3 py-1.5 text-xs rounded-md border border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 cursor-pointer';
      disconnectBtn.textContent = 'Disconnect';
      disconnectBtn.addEventListener('click', () => disconnectProvider(p.id));
      btnGroup.appendChild(disconnectBtn);
    } else if (p.hasCreds) {
      const connectBtn = document.createElement('button');
      connectBtn.className = 'px-3 py-1.5 text-xs rounded-md bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 hover:bg-neutral-700 dark:hover:bg-neutral-300 cursor-pointer';
      connectBtn.textContent = 'Connect';
      connectBtn.addEventListener('click', () => connectProvider(p.id));
      btnGroup.appendChild(connectBtn);
    } else {
      const setupBtn = document.createElement('button');
      setupBtn.className = 'px-3 py-1.5 text-xs rounded-md border border-neutral-300 dark:border-neutral-600 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 cursor-pointer';
      setupBtn.textContent = 'Setup';
      setupBtn.addEventListener('click', () => {
        // Toggle form visibility
        const existing = card.querySelector('.credential-form');
        if (existing) {
          existing.remove();
          return;
        }
        const form = createCredentialForm(p.id, () => renderCloudModal());
        if (form) {
          form.classList.add('credential-form');
          card.appendChild(form);
        }
      });
      btnGroup.appendChild(setupBtn);
    }

    header.appendChild(btnGroup);
    card.appendChild(header);
    content.appendChild(card);
  });

  // Browse section for each connected provider
  const schemeMap = { gdrive: 'gdrive://', s3: 's3://', dropbox: 'dropbox://', onedrive: 'onedrive://' };
  const connectedProviders = providers.filter(p => p.connected);

  connectedProviders.forEach(cp => {
    const browseSection = document.createElement('div');
    browseSection.className = 'mt-4 pt-4 border-t border-neutral-200 dark:border-neutral-700';

    const browseTitle = document.createElement('div');
    browseTitle.className = 'text-xs font-medium uppercase tracking-wide text-neutral-500 mb-2';
    browseTitle.textContent = 'Browse ' + cp.name;
    browseSection.appendChild(browseTitle);

    const pathDisplay = document.createElement('div');
    pathDisplay.className = 'text-xs text-neutral-500 mb-2 font-mono';
    pathDisplay.textContent = schemeMap[cp.id] || (cp.id + '://');
    browseSection.appendChild(pathDisplay);

    const folderList = document.createElement('div');
    folderList.className = 'flex flex-col gap-1 max-h-48 overflow-y-auto';
    browseSection.appendChild(folderList);

    let currentPath = '';

    async function renderFolders(path) {
      currentPath = path;
      const scheme = schemeMap[cp.id] || (cp.id + '://');
      pathDisplay.textContent = scheme + (path || '');
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

      const folders = await browseProvider(cp.id, path);
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

    const useBtn = document.createElement('button');
    useBtn.className = 'mt-2 w-full py-1.5 text-xs font-medium rounded-md bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 hover:bg-neutral-700 dark:hover:bg-neutral-300 cursor-pointer';
    useBtn.textContent = 'Use This Folder';
    useBtn.addEventListener('click', () => {
      const dirInput = document.getElementById('dirInput');
      if (dirInput) {
        const scheme = schemeMap[cp.id] || (cp.id + '://');
        dirInput.value = scheme + currentPath;
      }
      closeCloudModal();
    });
    browseSection.appendChild(useBtn);

    content.appendChild(browseSection);
    renderFolders('');
  });
}

export { closeCloudModal };
