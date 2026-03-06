export function isDark() {
  return document.documentElement.classList.contains('dark');
}

export function btnClass() {
  return 'h-8 px-3 text-sm rounded-md border cursor-pointer min-w-[40px] text-center transition-colors ' +
    (isDark()
      ? 'border-neutral-700 bg-neutral-800 text-neutral-400 hover:bg-neutral-700 hover:text-neutral-200'
      : 'border-neutral-300 bg-neutral-100 text-neutral-600 hover:bg-neutral-200 hover:text-neutral-900');
}

export function btnSelectedClass() {
  return 'h-8 px-3 text-sm rounded-md border font-medium min-w-[40px] text-center cursor-pointer transition-colors ' +
    (isDark()
      ? 'border-neutral-100 bg-neutral-100 text-neutral-900'
      : 'border-neutral-900 bg-neutral-900 text-white');
}

export function btnAddClass() {
  return 'h-8 px-3 text-sm rounded-md border border-dashed cursor-pointer bg-transparent transition-colors ' +
    (isDark()
      ? 'border-neutral-700 text-neutral-600 hover:border-neutral-500 hover:text-neutral-400'
      : 'border-neutral-300 text-neutral-400 hover:border-neutral-500 hover:text-neutral-600');
}

export function setupThemeToggle(onThemeChange) {
  const btn = document.getElementById('btnTheme');
  const icon = document.getElementById('themeIcon');
  function updateIcon() {
    icon.textContent = isDark() ? '\u2600' : '\u263D';
  }
  updateIcon();
  btn.addEventListener('click', () => {
    document.documentElement.classList.toggle('dark');
    localStorage.setItem('theme', isDark() ? 'dark' : 'light');
    updateIcon();
    if (onThemeChange) onThemeChange();
  });
}
