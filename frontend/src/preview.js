import { state } from './state.js';
import { mruBump } from './utils.js';

export function getBaseName(filename) {
  const dotIdx = filename.lastIndexOf('.');
  const base = filename.substring(0, dotIdx);
  const dblIdx = base.indexOf('__');
  return dblIdx === -1 ? base : base.substring(0, dblIdx);
}

export function updatePreview() {
  const previewEl = document.getElementById('previewName');
  const applyBtn = document.getElementById('btnApply');

  if (state.currentIndex < 0 || !state.projectConfig) {
    previewEl.textContent = '-';
    return;
  }

  const original = state.videos[state.currentIndex];
  const dotIdx = original.lastIndexOf('.');
  const ext = original.substring(dotIdx + 1);
  const baseName = getBaseName(original);

  let format = state.projectConfig.outputFormat || '{basename}__{groups}.{ext}';

  state.projectConfig.groups.forEach(group => {
    const sel = state.groupSelections[group.key];
    let val = '';
    if (sel instanceof Set && sel.size > 0) {
      const sep = group.separator || '_';
      val = [...sel].map(v => (group.prefix || '') + v).join(sep);
    } else if (typeof sel === 'string' && sel) {
      val = (group.prefix || '') + sel;
    }
    format = format.replace('{' + group.key + '}', val);
  });

  format = format.replace('{basename}', baseName);
  format = format.replace('{ext}', ext);
  format = format.replace('{date}', new Date().toISOString().slice(0, 10));
  format = format.replace('{original}', original);

  // Clean up empty sections
  let result = format.replace(/__+/g, '__').replace(/^__/, '').replace(/__\./g, '.');

  previewEl.textContent = result;
  applyBtn.disabled = (result === original);
}

// Parse existing annotations from filename — updates state only, does not render
export function parseAnnotations(filename) {
  if (!state.projectConfig || !state.projectConfig.groups) return;

  const dotIdx = filename.lastIndexOf('.');
  const base = filename.substring(0, dotIdx);
  const dblIdx = base.indexOf('__');
  if (dblIdx === -1) return;

  const annotationStr = base.substring(dblIdx + 2);
  const segments = annotationStr.split('__');

  for (const seg of segments) {
    let matched = false;

    // Check prefixed groups first (e.g. S88 → Subject group with prefix "S")
    for (const group of state.projectConfig.groups) {
      if (group.prefix && seg.startsWith(group.prefix)) {
        const val = seg.substring(group.prefix.length);
        if (group.options.includes(val)) {
          if (group.type === 'multi-select') {
            state.groupSelections[group.key].add(val);
            if (state.mruByGroup[group.key]) mruBump(state.mruByGroup[group.key], val);
          } else {
            state.groupSelections[group.key] = val;
          }
          matched = true;
          break;
        }
      }
    }
    if (matched) continue;

    // Check single-select exact matches (e.g. "great" → Quality group)
    for (const group of state.projectConfig.groups) {
      if (group.type === 'single-select' && !group.prefix && group.options.includes(seg)) {
        state.groupSelections[group.key] = seg;
        matched = true;
        break;
      }
    }
    if (matched) continue;

    // Check multi-select underscore-separated (e.g. "tag1_tag2" → Tags group)
    const parts = seg.split('_');
    for (const group of state.projectConfig.groups) {
      if (group.type === 'multi-select' && !group.prefix) {
        for (const p of parts) {
          if (group.options.includes(p)) {
            state.groupSelections[group.key].add(p);
          }
        }
      }
    }
  }
}
