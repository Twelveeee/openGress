import React from 'react';

export default function TopBar({ onOpen }) {
  return (
    <header className="top-bar">
      <div className="player-chip">
        <div className="player-avatar">T</div>
        <div className="player-info">
          <span className="player-name">Twelveeee</span>
          <span className="player-level">L13 · RES</span>
          <div className="xm-bar" role="img" aria-label="XM 4,023 / 5,000">
            <div className="xm-bar-track">
              <div className="xm-bar-fill" style={{ width: '80%' }} />
            </div>
            <div className="xm-bar-text">
              <span>4,023</span>
              <small>/ 5,000 XM</small>
            </div>
          </div>
        </div>
      </div>
      <div className="top-actions">
        <button className="icon-btn" onClick={() => onOpen('layers')} title="图层">
          🗺️
        </button>
        <button className="icon-btn" onClick={() => onOpen('settings')} title="设置">
          ⚙️
        </button>
        <button className="icon-btn" onClick={() => onOpen('inventory')} title="库存">
          🎒
        </button>
        <button className="icon-btn" onClick={() => onOpen('attack')} title="攻击">
          🔥
        </button>
        <button className="icon-btn" onClick={() => onOpen('leaderboard')} title="排行榜">
          🏆
        </button>
      </div>
    </header>
  );
}
