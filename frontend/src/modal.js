let modalCallback = null;

export function setupModal() {
  const confirm = document.getElementById('modalConfirm');
  const cancel = document.getElementById('modalCancel');
  const input = document.getElementById('modalInput');

  confirm.addEventListener('click', () => {
    const val = input.value.trim();
    const cb = modalCallback;
    hideModal();
    if (cb && val) cb(val);
  });
  cancel.addEventListener('click', hideModal);
  input.addEventListener('keydown', e => {
    if (e.key === 'Enter') confirm.click();
    if (e.key === 'Escape') hideModal();
  });
}

export function showModal(title, callback) {
  const modal = document.getElementById('modal');
  document.getElementById('modalTitle').textContent = title;
  document.getElementById('modalInput').value = '';
  modal.classList.remove('hidden');
  modal.classList.add('flex');
  document.getElementById('modalInput').focus();
  modalCallback = callback;
}

function hideModal() {
  const modal = document.getElementById('modal');
  modal.classList.add('hidden');
  modal.classList.remove('flex');
  modalCallback = null;
}

export function isModalOpen() {
  return !document.getElementById('modal').classList.contains('hidden');
}
