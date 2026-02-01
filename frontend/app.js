const modals = {
  portal: document.getElementById('modal-portal'),
  player: document.getElementById('modal-player'),
  inventory: document.getElementById('modal-inventory'),
  attack: document.getElementById('modal-attack'),
  leaderboard: document.getElementById('modal-leaderboard'),
  settings: document.getElementById('modal-settings'),
  layers: document.getElementById('modal-layers')
};

function openModal(name) {
  const modal = modals[name];
  if (!modal) return;
  modal.classList.add('is-open');
  modal.setAttribute('aria-hidden', 'false');
}

function closeModal(modal) {
  modal.classList.remove('is-open');
  modal.setAttribute('aria-hidden', 'true');
}

function handleModalOpen(event) {
  const trigger = event.target.closest('[data-open]');
  if (!trigger) return;
  const target = trigger.dataset.open;
  openModal(target);
}

function bindModalClose() {
  document.querySelectorAll('[data-close]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const modal = btn.closest('.modal');
      if (modal) closeModal(modal);
    });
  });

  document.querySelectorAll('.modal').forEach((modal) => {
    modal.addEventListener('click', (event) => {
      if (event.target === modal) closeModal(modal);
    });
  });
}

function init() {
  document.body.addEventListener('click', handleModalOpen);
  bindModalClose();
}

init();
