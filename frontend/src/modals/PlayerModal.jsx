import React from 'react';
import Modal from '../components/Modal.jsx';

export default function PlayerModal({ open, onClose }) {
  return (
    <Modal open={open} onClose={onClose} className="profile-card">
      <header className="modal-header">
        <h3>Agent Profile</h3>
      </header>
      <div className="profile-body">
        <div className="profile-top">
          <div className="player-avatar large">T</div>
          <div>
            <p className="player-name">Twelveeee</p>
            <p className="muted">L13 · Resistance</p>
          </div>
          <div className="ap-bar">
            <span>4,023,289 / 5,000,000 AP</span>
            <div className="bar">
              <div className="fill" />
            </div>
          </div>
        </div>
        <div className="badge-grid">
          <span className="badge">Eureka</span>
          <span className="badge">Seer</span>
          <span className="badge">Illuminator</span>
          <span className="badge">Purifier</span>
          <span className="badge">Builder</span>
          <span className="badge">Recharger</span>
          <span className="badge">Explorer</span>
          <span className="badge">SpecOps</span>
        </div>
      </div>
    </Modal>
  );
}
