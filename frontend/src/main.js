import { state } from './state.js';
import { fetchFiles, loadConfig, renameFile, saveSession, loadSession, loadUserSettings, openFolder } from './api.js';
import { setupThemeToggle } from './theme.js';
import { setupModal, isModalOpen } from './modal.js';
import { renderAllGroups } from './groups.js';
import { updatePreview, parseAnnotations } from './preview.js';
import { setupFileList, renderFileList } from './fileList.js';
import { setupConfigEditor } from './configEditor.js';

function parseFileList(data) {
  state.fileMeta = {};
  if (!data || data.error) { state.files = []; return; }
  state.files = data.map(v => v.name);
  data.forEach(v => { state.fileMeta[v.name] = { size: v.size, modified: v.modified }; });
}

function clearSelections() {
  if (state.projectConfig && state.projectConfig.groups) {
    state.projectConfig.groups.forEach(g => {
      state.groupSelections[g.key] = g.type === 'multi-select' ? new Set() : null;
    });
  }
}

function resetSelections() {
  clearSelections();
  renderAllGroups();
  updatePreview();
}

function selectFile(index) {
  state.currentIndex = index;
  const name = state.files[index];
  const player = document.getElementById('videoPlayer');
  const noMedia = document.getElementById('noVideo');

  clearSelections();
  parseAnnotations(name);
  renderAllGroups();
  updatePreview();

  noMedia.style.display = 'none';
  player.style.display = '';
  document.getElementById('currentFileName').textContent = name;
  player.src = '/api/media?dir=' + encodeURIComponent(state.currentDir) + '&file=' + encodeURIComponent(name);
  player.load();
  player.play().catch(() => {});

  document.getElementById('btnPrev').disabled = index === 0;
  document.getElementById('btnNext').disabled = index >= state.files.length - 1;
  document.getElementById('navInfo').textContent = (index + 1) + ' / ' + state.files.length;

  renderFileList();

  const rows = document.getElementById('fileListBody').querySelectorAll('tr');
  if (rows[index]) rows[index].scrollIntoView({ block: 'nearest' });

  saveSession();
}

async function loadFolder() {
  const dirInput = document.getElementById('dirInput');
  const dir = dirInput.value.trim();
  if (!dir) return;
  state.currentDir = dir;

  try {
    const data = await fetchFiles(dir);
    if (data.error) { alert(data.error); return; }
    parseFileList(data);
    document.getElementById('fileCounter').textContent = state.files.length + ' files';
    await loadConfig();
    renderAllGroups();
    renderFileList();
    if (state.files.length > 0) {
      selectFile(0);
    } else {
      state.currentIndex = -1;
      document.getElementById('noVideo').style.display = '';
      document.getElementById('videoPlayer').style.display = 'none';
      document.getElementById('currentFileName').textContent = 'No files found';
    }
    saveSession();
  } catch (e) {
    alert('Failed to load folder: ' + e.message);
  }
}

async function restoreSession() {
  try {
    const session = await loadSession();

    // Restore MRU (handle old and new session format)
    if (session.mruByGroup) {
      state.mruByGroup = session.mruByGroup;
    } else {
      if (session.mruSubjects) state.mruByGroup['S'] = session.mruSubjects;
      if (session.mruTags) state.mruByGroup['tags'] = session.mruTags;
    }

    if (session.dir) {
      document.getElementById('dirInput').value = session.dir;
      state.currentDir = session.dir;
      const data = await fetchFiles(session.dir);
      if (data.error) return;
      parseFileList(data);
      document.getElementById('fileCounter').textContent = state.files.length + ' files';
      await loadConfig();
      renderAllGroups();
      renderFileList();
      if (state.files.length > 0) {
        let idx = 0;
        if (session.file) {
          const found = state.files.indexOf(session.file);
          if (found >= 0) idx = found;
        }
        selectFile(idx);
      }
    }
  } catch (e) {}
}

async function applyRename() {
  if (state.currentIndex < 0) return;
  const player = document.getElementById('videoPlayer');
  const oldName = state.files[state.currentIndex];
  const newName = document.getElementById('previewName').textContent;
  if (oldName === newName) return;

  // Release file lock before renaming (Windows)
  player.pause();
  player.removeAttribute('src');
  player.load();
  await new Promise(r => setTimeout(r, 100));

  try {
    const data = await renameFile(oldName, newName);
    if (data.error) { alert('Rename failed: ' + data.error); return; }
    state.files[state.currentIndex] = newName;
    saveSession();
    if (state.currentIndex < state.files.length - 1) {
      selectFile(state.currentIndex + 1);
    } else {
      selectFile(state.currentIndex);
    }
  } catch (e) {
    alert('Rename failed: ' + e.message);
  }
}

function setupSplitter() {
  const handle = document.getElementById('splitHandle');
  const fileList = document.getElementById('fileList');
  let dragging = false, startY = 0, startH = 0;

  handle.addEventListener('mousedown', e => {
    dragging = true; startY = e.clientY; startH = fileList.offsetHeight;
    document.body.style.cursor = 'row-resize';
    document.body.style.userSelect = 'none';
    e.preventDefault();
  });
  document.addEventListener('mousemove', e => {
    if (!dragging) return;
    fileList.style.height = Math.max(50, startH - (e.clientY - startY)) + 'px';
  });
  document.addEventListener('mouseup', () => {
    if (!dragging) return;
    dragging = false;
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  });
}

function init() {
  const dirInput = document.getElementById('dirInput');
  const btnPrev = document.getElementById('btnPrev');
  const btnNext = document.getElementById('btnNext');
  const btnApply = document.getElementById('btnApply');

  setupThemeToggle(() => renderAllGroups());
  setupModal();
  setupFileList(selectFile);
  setupConfigEditor();
  setupSplitter();

  document.getElementById('btnLoad').addEventListener('click', loadFolder);
  dirInput.addEventListener('keydown', e => { if (e.key === 'Enter') loadFolder(); });
  document.getElementById('btnOpenFolder').addEventListener('click', () => {
    const dir = dirInput.value.trim();
    if (dir) openFolder(dir);
  });

  btnPrev.addEventListener('click', () => { if (state.currentIndex > 0) selectFile(state.currentIndex - 1); });
  btnNext.addEventListener('click', () => { if (state.currentIndex < state.files.length - 1) selectFile(state.currentIndex + 1); });

  btnApply.addEventListener('click', applyRename);
  document.getElementById('btnSkip').addEventListener('click', () => {
    if (state.currentIndex < state.files.length - 1) selectFile(state.currentIndex + 1);
  });
  document.getElementById('btnReset').addEventListener('click', resetSelections);

  document.addEventListener('keydown', e => {
    if (isModalOpen()) return;
    if (document.activeElement === dirInput) return;
    if (document.activeElement && document.activeElement.closest('#configEditor')) return;
    if (e.key === 'ArrowLeft' && !btnPrev.disabled) { e.preventDefault(); btnPrev.click(); }
    if (e.key === 'ArrowRight' && !btnNext.disabled) { e.preventDefault(); btnNext.click(); }
    if (e.key === 'Enter' && !btnApply.disabled) { e.preventDefault(); btnApply.click(); }
    if (e.key === 'Escape') resetSelections();
  });

  loadUserSettings().then(() => restoreSession());
}

init();
