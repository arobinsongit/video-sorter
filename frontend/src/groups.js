import { state } from './state.js';
import { btnClass, btnSelectedClass, btnAddClass } from './theme.js';
import { showModal } from './modal.js';
import { saveConfig } from './api.js';
import { clearChildren, mruBump, mruSort } from './utils.js';
import { updatePreview } from './preview.js';

export function renderAllGroups() {
  const container = document.getElementById('groupsContainer');
  clearChildren(container);
  if (!state.projectConfig || !state.projectConfig.groups) return;
  state.projectConfig.groups.forEach(group => {
    container.appendChild(renderGroup(group));
  });
}

function renderGroup(group) {
  const div = document.createElement('div');
  div.className = 'rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900 p-3';

  // Header
  const header = document.createElement('div');
  header.className = 'flex justify-between items-center mb-2.5';

  const label = document.createElement('span');
  label.className = 'text-[11px] font-medium uppercase tracking-wide text-neutral-500';
  label.textContent = group.name;

  const right = document.createElement('div');
  right.className = 'flex items-center gap-2';

  const valueSpan = document.createElement('span');
  valueSpan.className = 'text-[11px] font-medium text-neutral-900 dark:text-neutral-100';
  updateGroupValueDisplay(valueSpan, group);

  const clearBtn = document.createElement('button');
  clearBtn.className = 'w-5 h-5 flex items-center justify-center rounded-full bg-neutral-200 dark:bg-neutral-700 text-neutral-500 dark:text-neutral-400 hover:bg-red-100 hover:text-red-500 dark:hover:bg-red-900 dark:hover:text-red-400 transition-colors text-xs leading-none';
  clearBtn.textContent = '\u00D7';
  clearBtn.addEventListener('click', () => {
    state.groupSelections[group.key] = group.type === 'multi-select' ? new Set() : null;
    renderAllGroups();
    updatePreview();
  });

  right.appendChild(valueSpan);
  right.appendChild(clearBtn);
  header.appendChild(label);
  header.appendChild(right);
  div.appendChild(header);

  // Content
  if (group.type === 'single-select' && group.inputType === 'slider') {
    renderSliderGroup(div, group);
  } else if (group.type === 'single-select') {
    renderRadioGroup(div, group);
  } else {
    renderMultiSelectGroup(div, group);
  }

  return div;
}

function updateGroupValueDisplay(el, group) {
  const sel = state.groupSelections[group.key];
  if (sel instanceof Set) {
    el.textContent = sel.size > 0 ? [...sel].join(', ') : '';
  } else {
    el.textContent = sel || '';
  }
}

function renderMultiSelectGroup(container, group) {
  const grid = document.createElement('div');
  grid.className = 'flex flex-wrap gap-1.5';

  const mru = state.mruByGroup[group.key] || [];
  const sorted = mruSort(group.options, mru);
  const sel = state.groupSelections[group.key];

  sorted.forEach(val => {
    const btn = document.createElement('button');
    btn.className = sel.has(val) ? btnSelectedClass() : btnClass();
    btn.textContent = val;
    btn.addEventListener('click', () => {
      if (sel.has(val)) {
        sel.delete(val);
      } else {
        sel.add(val);
        if (state.mruByGroup[group.key]) mruBump(state.mruByGroup[group.key], val);
      }
      renderAllGroups();
      updatePreview();
    });
    grid.appendChild(btn);
  });

  if (group.allowCustom) {
    const addBtn = document.createElement('button');
    addBtn.className = btnAddClass();
    addBtn.textContent = '+';
    addBtn.addEventListener('click', () => showModal('Add ' + group.name, val => {
      if (group.inputType === 'text') val = val.toLowerCase().replace(/\s+/g, '-');
      if (val && !group.options.includes(val)) {
        group.options.push(val);
      }
      if (val) {
        sel.add(val);
        if (state.mruByGroup[group.key]) mruBump(state.mruByGroup[group.key], val);
        saveConfig();
        renderAllGroups();
        updatePreview();
      }
    }));
    grid.appendChild(addBtn);
  }

  container.appendChild(grid);
}

function renderSliderGroup(container, group) {
  const wrapper = document.createElement('div');
  wrapper.className = 'px-2';

  const slider = document.createElement('input');
  slider.type = 'range';
  slider.min = '0';
  slider.max = String(group.options.length - 1);
  slider.step = '1';

  const currentVal = state.groupSelections[group.key];
  const currentIdx = currentVal ? group.options.indexOf(currentVal) : -1;
  slider.value = currentIdx >= 0 ? String(currentIdx) : String(Math.floor(group.options.length / 2));

  slider.className = 'w-full h-1.5 appearance-none rounded-full bg-neutral-200 dark:bg-neutral-700 cursor-pointer ' +
    '[&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-4 [&::-webkit-slider-thumb]:h-4 ' +
    '[&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-neutral-900 [&::-webkit-slider-thumb]:dark:bg-neutral-100 ' +
    '[&::-webkit-slider-thumb]:shadow-sm [&::-webkit-slider-thumb]:cursor-pointer ' +
    '[&::-moz-range-thumb]:w-4 [&::-moz-range-thumb]:h-4 [&::-moz-range-thumb]:rounded-full ' +
    '[&::-moz-range-thumb]:bg-neutral-900 [&::-moz-range-thumb]:dark:bg-neutral-100 ' +
    '[&::-moz-range-thumb]:border-0 [&::-moz-range-thumb]:cursor-pointer';

  const labels = document.createElement('div');
  labels.className = 'flex justify-between mt-1.5';

  function updateLabels(activeIdx) {
    clearChildren(labels);
    const len = group.options.length;
    group.options.forEach((opt, i) => {
      const span = document.createElement('span');
      span.className = 'text-sm text-center w-12';
      if (i === 0) span.className += ' -ml-4';
      if (i === len - 1) span.className += ' -mr-4';

      if (i === 0) span.classList.add('text-red-500');
      else if (i === len - 1) span.classList.add('text-green-500', 'dark:text-green-400');
      else if (i > len / 2) span.classList.add('text-green-700', 'dark:text-green-500');
      else span.classList.add('text-neutral-400');

      if (i === activeIdx) {
        span.classList.add('font-bold', 'scale-110');
      }
      span.textContent = opt.charAt(0).toUpperCase() + opt.slice(1);
      labels.appendChild(span);
    });
  }

  slider.addEventListener('input', () => {
    const idx = parseInt(slider.value);
    state.groupSelections[group.key] = group.options[idx];
    updateLabels(idx);
    updatePreview();
  });

  updateLabels(currentIdx);
  wrapper.appendChild(slider);
  wrapper.appendChild(labels);
  container.appendChild(wrapper);
}

function renderRadioGroup(container, group) {
  const grid = document.createElement('div');
  grid.className = 'flex flex-wrap gap-1.5';

  const sel = state.groupSelections[group.key];

  group.options.forEach(val => {
    const btn = document.createElement('button');
    btn.className = (sel === val) ? btnSelectedClass() : btnClass();
    btn.textContent = val;
    btn.addEventListener('click', () => {
      state.groupSelections[group.key] = (sel === val) ? null : val;
      renderAllGroups();
      updatePreview();
    });
    grid.appendChild(btn);
  });

  if (group.allowCustom) {
    const addBtn = document.createElement('button');
    addBtn.className = btnAddClass();
    addBtn.textContent = '+';
    addBtn.addEventListener('click', () => showModal('Add ' + group.name, val => {
      if (val && !group.options.includes(val)) {
        group.options.push(val);
      }
      if (val) {
        state.groupSelections[group.key] = val;
        saveConfig();
        renderAllGroups();
        updatePreview();
      }
    }));
    grid.appendChild(addBtn);
  }

  container.appendChild(grid);
}
