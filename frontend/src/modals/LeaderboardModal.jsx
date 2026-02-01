import React from 'react';
import Modal from '../components/Modal.jsx';

export default function LeaderboardModal({ open, onClose }) {
  return (
    <Modal open={open} onClose={onClose} className="leaderboard-card">
      <header className="modal-header">
        <h3>排行榜</h3>
      </header>
      <div className="leaderboard-body">
        <div className="tabs">
          <button className="tab active">全服</button>
          <button className="tab">Resistance</button>
          <button className="tab">Enlightened</button>
        </div>
        <div className="leaderboard-row">
          <span>#1</span>
          <span>AgentNova</span>
          <span>L10</span>
          <span className="res">RES</span>
          <span>2.4M AP</span>
        </div>
        <div className="leaderboard-row">
          <span>#2</span>
          <span>Cipher</span>
          <span>L9</span>
          <span className="enl">ENL</span>
          <span>2.1M AP</span>
        </div>
      </div>
    </Modal>
  );
}
