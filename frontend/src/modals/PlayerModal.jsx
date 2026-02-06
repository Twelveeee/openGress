import React from 'react';
import Modal from '../components/Modal.jsx';

function factionLabel(faction) {
  const key = String(faction || '').toUpperCase();
  if (key === 'RESISTANCE') return 'Resistance';
  if (key === 'ENLIGHTENED') return 'Enlightened';
  return 'Neutral';
}

export default function PlayerModal({ open, onClose, player, ap }) {
  return (
    <Modal open={open} onClose={onClose} className="profile-card">
      <header className="modal-header">
        <h3>Agent Profile</h3>
      </header>
      <div className="profile-body">
        <div className="profile-top">
          <div className="player-avatar large">{String(player?.username || 'A').slice(0, 1).toUpperCase()}</div>
          <div>
            <p className="player-name">{player?.username || 'Agent'}</p>
            <p className="muted">L{player?.level || 1} · {factionLabel(player?.faction)}</p>
          </div>
          <div className="ap-bar">
            <span>{ap?.text || `${player?.ap || 0} AP`}</span>
            <div className="bar">
              <div className="fill" style={{ width: `${ap?.percent || 0}%` }} />
            </div>
          </div>
        </div>
        <div className="badge-grid">
          <span className="badge">XM: {player?.xm || 0}</span>
          <span className="badge">Max XM: {player?.maxXm || 0}</span>
          <span className="badge">Auto Hack: {player?.autoHack ? 'ON' : 'OFF'}</span>
        </div>
      </div>
    </Modal>
  );
}
