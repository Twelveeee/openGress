import React from 'react';
import Modal from '../components/Modal.jsx';

export default function LeaderboardModal({ open, onClose, rows = [] }) {
  return (
    <Modal open={open} onClose={onClose} className="leaderboard-card">
      <header className="modal-header">
        <h3>排行榜</h3>
      </header>
      <div className="leaderboard-body">
        <div className="tabs">
          <button className="tab active">全服</button>
        </div>
        {rows.length ? (
          rows.slice(0, 20).map((row, idx) => (
            <div key={row.id || row.playerId || idx} className="leaderboard-row">
              <span>#{idx + 1}</span>
              <span>{row.username || row.name || 'Agent'}</span>
              <span>L{row.level || 1}</span>
              <span className={String(row.faction || '').startsWith('RES') ? 'res' : 'enl'}>
                {String(row.faction || '').slice(0, 3)}
              </span>
              <span>{Number(row.ap || 0).toLocaleString('en-US')} AP</span>
            </div>
          ))
        ) : (
          <div className="leaderboard-row">
            <span>#-</span>
            <span>暂无数据</span>
            <span>L-</span>
            <span>---</span>
            <span>0 AP</span>
          </div>
        )}
      </div>
    </Modal>
  );
}
