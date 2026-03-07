import { state } from './state.js';
import { saveConfig } from './api.js';
import { renderAllGroups } from './groups.js';
import { updatePreview } from './preview.js';
import { clearChildren } from './utils.js';

export function setupConfigEditor() {
  const btn = document.getElementById('btnSettings');
  const closeBtn = document.getElementById('configEditorClose');
  if (btn) btn.addEventListener('click', openConfigEditor);
  if (closeBtn) closeBtn.addEventListener('click', closeConfigEditor);
}

function openConfigEditor() {
  if (!state.projectConfig) return;
  const overlay = document.getElementById('configEditor');
  overlay.classList.remove('hidden');
  overlay.classList.add('flex');
  renderConfigEditor();
}

function closeConfigEditor() {
  const overlay = document.getElementById('configEditor');
  overlay.classList.add('hidden');
  overlay.classList.remove('flex');
  saveConfig();
  renderAllGroups();
  updatePreview();
}

function renderConfigEditor() {
  const content = document.getElementById('configEditorContent');
  clearChildren(content);
  const cfg = state.projectConfig;

  // Output Format
  content.appendChild(makeSection('Output Format', () => {
    const div = document.createElement('div');
    const input = makeInput(cfg.outputFormat, val => { cfg.outputFormat = val; updatePreview(); });
    input.placeholder = '{basename}__{S}__{tags}__{quality}.{ext}';
    div.appendChild(input);
    const help = document.createElement('div');
    help.className = 'text-[10px] text-neutral-400 mt-1';
    help.textContent = 'Tokens: {basename}, {ext}, {date}, {original}, and group keys like {S}, {tags}, {quality}';
    div.appendChild(help);
    return div;
  }));

  // Output Mode
  content.appendChild(makeSection('Output Mode', () => {
    const div = document.createElement('div');
    div.className = 'flex gap-2';
    ['rename', 'move', 'copy'].forEach(mode => {
      const btn = document.createElement('button');
      const isActive = (cfg.outputMode || 'rename') === mode;
      btn.className = isActive
        ? 'px-3 py-1.5 text-sm rounded-md bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 font-medium cursor-pointer'
        : 'px-3 py-1.5 text-sm rounded-md border border-neutral-300 dark:border-neutral-700 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800 cursor-pointer';
      btn.textContent = mode.charAt(0).toUpperCase() + mode.slice(1);
      btn.addEventListener('click', () => { cfg.outputMode = mode; renderConfigEditor(); });
      div.appendChild(btn);
    });
    return div;
  }));

  // Output Folder (only for move/copy)
  if (cfg.outputMode === 'move' || cfg.outputMode === 'copy') {
    content.appendChild(makeSection('Output Folder', () => {
      const input = makeInput(cfg.outputFolder || '', val => { cfg.outputFolder = val; });
      input.placeholder = 'Path (relative to media folder or absolute)';
      return input;
    }));
  }

  // Groups
  content.appendChild(makeSection('Metadata Groups', () => {
    const div = document.createElement('div');
    div.className = 'flex flex-col gap-3';

    cfg.groups.forEach((group, idx) => {
      const card = document.createElement('div');
      card.className = 'border border-neutral-200 dark:border-neutral-700 rounded-md p-3';

      // Group header
      const header = document.createElement('div');
      header.className = 'flex items-center gap-2 mb-2';
      const nameInput = makeInput(group.name, val => { group.name = val; });
      nameInput.classList.add('flex-1');
      nameInput.placeholder = 'Group name';
      header.appendChild(nameInput);

      const removeBtn = document.createElement('button');
      removeBtn.className = 'text-red-500 hover:text-red-700 text-sm px-2 cursor-pointer';
      removeBtn.textContent = 'Remove';
      removeBtn.addEventListener('click', () => {
        cfg.groups.splice(idx, 1);
        delete state.groupSelections[group.key];
        delete state.mruByGroup[group.key];
        renderConfigEditor();
      });
      header.appendChild(removeBtn);
      card.appendChild(header);

      // Key
      card.appendChild(makeRow('Key', makeInput(group.key, val => { group.key = val; })));

      // Type
      card.appendChild(makeRow('Type', makeSelect(
        [{ value: 'multi-select', label: 'Multi-select' }, { value: 'single-select', label: 'Single-select' }],
        group.type, val => { group.type = val; renderConfigEditor(); }
      )));

      // Input Type
      card.appendChild(makeRow('Input', makeSelect(
        [{ value: 'text', label: 'Text' }, { value: 'number', label: 'Number' }, { value: 'slider', label: 'Slider' }],
        group.inputType, val => { group.inputType = val; renderConfigEditor(); }
      )));

      // Prefix & Separator
      const prefixRow = document.createElement('div');
      prefixRow.className = 'flex gap-2 mt-1';
      prefixRow.appendChild(makeRow('Prefix', makeInput(group.prefix || '', val => { group.prefix = val; })));
      prefixRow.appendChild(makeRow('Sep', makeInput(group.separator || '', val => { group.separator = val; })));
      card.appendChild(prefixRow);

      // Options
      const optionsLabel = document.createElement('div');
      optionsLabel.className = 'text-[10px] text-neutral-500 mt-2 mb-1';
      optionsLabel.textContent = 'Options (one per line)';
      card.appendChild(optionsLabel);

      const textarea = document.createElement('textarea');
      textarea.className = 'w-full h-20 px-2 py-1 text-xs bg-neutral-50 dark:bg-neutral-950 border border-neutral-300 dark:border-neutral-700 rounded-md text-neutral-900 dark:text-neutral-100 font-mono resize-y focus:outline-none focus:ring-1 focus:ring-neutral-400';
      textarea.value = group.options.join('\n');
      textarea.addEventListener('change', () => {
        group.options = textarea.value.split('\n').map(s => s.trim()).filter(s => s);
      });
      card.appendChild(textarea);

      // Allow Custom
      const customRow = document.createElement('label');
      customRow.className = 'flex items-center gap-2 mt-2 text-xs text-neutral-600 dark:text-neutral-400 cursor-pointer';
      const checkbox = document.createElement('input');
      checkbox.type = 'checkbox';
      checkbox.checked = group.allowCustom;
      checkbox.addEventListener('change', () => { group.allowCustom = checkbox.checked; });
      customRow.appendChild(checkbox);
      customRow.appendChild(document.createTextNode('Allow custom values'));
      card.appendChild(customRow);

      div.appendChild(card);
    });

    // Add group button
    const addBtn = document.createElement('button');
    addBtn.className = 'w-full py-2 text-sm rounded-md border border-dashed border-neutral-300 dark:border-neutral-700 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 hover:border-neutral-500 cursor-pointer transition-colors';
    addBtn.textContent = '+ Add Group';
    addBtn.addEventListener('click', () => {
      const key = 'group' + (cfg.groups.length + 1);
      cfg.groups.push({
        name: 'New Group', key, type: 'multi-select', inputType: 'text',
        options: [], allowCustom: true, separator: '_', prefix: ''
      });
      state.groupSelections[key] = new Set();
      state.mruByGroup[key] = [];
      renderConfigEditor();
    });
    div.appendChild(addBtn);

    return div;
  }));

  // Done button
  const closeBtn = document.createElement('button');
  closeBtn.className = 'w-full mt-4 py-2 text-sm font-medium rounded-md bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 hover:bg-neutral-700 dark:hover:bg-neutral-300 cursor-pointer transition-colors';
  closeBtn.textContent = 'Done';
  closeBtn.addEventListener('click', closeConfigEditor);
  content.appendChild(closeBtn);
}

function makeSection(title, contentFn) {
  const section = document.createElement('div');
  section.className = 'mb-4';
  const h = document.createElement('div');
  h.className = 'text-[11px] font-medium uppercase tracking-wide text-neutral-500 mb-2';
  h.textContent = title;
  section.appendChild(h);
  section.appendChild(contentFn());
  return section;
}

function makeInput(value, onChange) {
  const input = document.createElement('input');
  input.type = 'text';
  input.value = value;
  input.className = 'w-full h-8 px-2 text-sm bg-neutral-50 dark:bg-neutral-950 border border-neutral-300 dark:border-neutral-700 rounded-md text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-1 focus:ring-neutral-400';
  input.addEventListener('change', () => onChange(input.value));
  return input;
}

function makeSelect(options, current, onChange) {
  const select = document.createElement('select');
  select.className = 'h-8 px-2 text-sm bg-neutral-50 dark:bg-neutral-950 border border-neutral-300 dark:border-neutral-700 rounded-md text-neutral-900 dark:text-neutral-100 focus:outline-none';
  options.forEach(opt => {
    const o = document.createElement('option');
    o.value = opt.value;
    o.textContent = opt.label;
    o.selected = opt.value === current;
    select.appendChild(o);
  });
  select.addEventListener('change', () => onChange(select.value));
  return select;
}

function makeRow(label, inputEl) {
  const row = document.createElement('div');
  row.className = 'flex items-center gap-2 mt-1';
  const lbl = document.createElement('span');
  lbl.className = 'text-[10px] text-neutral-500 w-12 flex-shrink-0';
  lbl.textContent = label;
  row.appendChild(lbl);
  inputEl.classList.add('flex-1');
  row.appendChild(inputEl);
  return row;
}
