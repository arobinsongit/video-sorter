export function clearChildren(el) {
  while (el.firstChild) el.removeChild(el.firstChild);
}

export const $ = id => document.getElementById(id);

export function mruBump(arr, val) {
  const idx = arr.indexOf(val);
  if (idx > -1) arr.splice(idx, 1);
  arr.unshift(val);
}

export function mruSort(items, mru) {
  const order = new Map(mru.map((v, i) => [v, i]));
  return [...items].sort((a, b) => {
    const ai = order.has(a) ? order.get(a) : Infinity;
    const bi = order.has(b) ? order.get(b) : Infinity;
    if (ai !== bi) return ai - bi;
    return items.indexOf(a) - items.indexOf(b);
  });
}

export function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB';
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}
