import { state } from './state.js';

export async function fetchFiles(dir) {
  const resp = await fetch('/api/list?dir=' + encodeURIComponent(dir));
  return resp.json();
}

export async function loadConfig() {
  try {
    const resp = await fetch('/api/config?dir=' + encodeURIComponent(state.currentDir));
    state.projectConfig = await resp.json();
  } catch (e) {
    state.projectConfig = null;
  }
  initGroupSelections();
}

export function initGroupSelections() {
  state.groupSelections = {};
  if (!state.projectConfig || !state.projectConfig.groups) return;
  state.projectConfig.groups.forEach(g => {
    state.groupSelections[g.key] = g.type === 'multi-select' ? new Set() : null;
    if (!state.mruByGroup[g.key]) state.mruByGroup[g.key] = [];
  });
}

export async function saveConfig() {
  if (!state.currentDir || !state.projectConfig) return;
  await fetch('/api/config/save', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ dir: state.currentDir, config: state.projectConfig })
  }).catch(() => {});
}

export async function renameFile(oldName, newName) {
  const resp = await fetch('/api/rename', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      dir: state.currentDir,
      oldName,
      newName,
      outputMode: state.projectConfig?.outputMode || 'rename',
      outputFolder: state.projectConfig?.outputFolder || ''
    })
  });
  return resp.json();
}

export async function saveSession() {
  if (!state.currentDir) return;
  const file = state.currentIndex >= 0 ? state.files[state.currentIndex] : '';
  await fetch('/api/session/save', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ dir: state.currentDir, file, mruByGroup: state.mruByGroup })
  }).catch(() => {});
}

export async function loadSession() {
  try {
    const resp = await fetch('/api/session');
    return resp.json();
  } catch (e) {
    return {};
  }
}

export async function loadUserSettings() {
  try {
    const resp = await fetch('/api/user-settings');
    state.userSettings = await resp.json();
  } catch (e) {
    state.userSettings = {};
  }
}

export async function saveUserSettings() {
  await fetch('/api/user-settings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(state.userSettings)
  }).catch(() => {});
}

export function openFolder(dir) {
  fetch('/api/open-folder?dir=' + encodeURIComponent(dir));
}
