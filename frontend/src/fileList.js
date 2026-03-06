import { state } from './state.js';
import { clearChildren, formatSize } from './utils.js';
import { isDark } from './theme.js';

let sortCol = 'name', sortAsc = true;
let onSelectVideo = null;

export function setupFileList(selectVideoFn) {
  onSelectVideo = selectVideoFn;
  document.getElementById('colName').addEventListener('click', () => toggleSort('name'));
  document.getElementById('colSize').addEventListener('click', () => toggleSort('size'));
  document.getElementById('colDate').addEventListener('click', () => toggleSort('date'));
  document.getElementById('fileFilter').addEventListener('input', () => renderFileList());
  updateSortIndicators();
}

function toggleSort(col) {
  if (sortCol === col) sortAsc = !sortAsc;
  else { sortCol = col; sortAsc = true; }
  updateSortIndicators();
  renderFileList();
}

function updateSortIndicators() {
  const arrow = col => col === sortCol ? (sortAsc ? '\u25B2' : '\u25BC') : '';
  document.getElementById('colNameSort').textContent = arrow('name');
  document.getElementById('colSizeSort').textContent = arrow('size');
  document.getElementById('colDateSort').textContent = arrow('date');
}

function getFilteredSorted() {
  const filter = (document.getElementById('fileFilter').value || '').toLowerCase();
  let indices = state.videos.map((_, i) => i);
  if (filter) {
    indices = indices.filter(i => state.videos[i].toLowerCase().includes(filter));
  }
  indices.sort((a, b) => {
    const ma = state.videoMeta[state.videos[a]] || {};
    const mb = state.videoMeta[state.videos[b]] || {};
    let cmp = 0;
    if (sortCol === 'name') cmp = state.videos[a].toLowerCase().localeCompare(state.videos[b].toLowerCase());
    else if (sortCol === 'size') cmp = (ma.size || 0) - (mb.size || 0);
    else if (sortCol === 'date') cmp = (ma.modified || '').localeCompare(mb.modified || '');
    return sortAsc ? cmp : -cmp;
  });
  return indices;
}

export function renderFileList() {
  const tbody = document.getElementById('fileListBody');
  clearChildren(tbody);
  const indices = getFilteredSorted();
  indices.forEach(i => {
    const name = state.videos[i];
    const meta = state.videoMeta[name] || {};
    const active = i === state.currentIndex;
    const row = document.createElement('tr');
    row.className = 'cursor-pointer transition-colors ' +
      (active
        ? (isDark() ? 'bg-neutral-800 text-neutral-100' : 'bg-neutral-200 text-neutral-900')
        : (isDark() ? 'text-neutral-500 hover:bg-neutral-800 hover:text-neutral-300' : 'text-neutral-500 hover:bg-neutral-100 hover:text-neutral-800'));

    const nameTd = document.createElement('td');
    nameTd.className = 'px-3 py-1 truncate overflow-hidden';
    nameTd.textContent = name;

    const sizeTd = document.createElement('td');
    sizeTd.className = 'px-3 py-1 text-right whitespace-nowrap';
    sizeTd.textContent = meta.size ? formatSize(meta.size) : '';

    const dateTd = document.createElement('td');
    dateTd.className = 'px-3 py-1 text-right whitespace-nowrap';
    dateTd.textContent = meta.modified || '';

    row.appendChild(nameTd);
    row.appendChild(sizeTd);
    row.appendChild(dateTd);
    row.addEventListener('click', () => { if (onSelectVideo) onSelectVideo(i); });
    tbody.appendChild(row);
  });
}
