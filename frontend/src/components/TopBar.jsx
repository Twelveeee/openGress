import React from 'react';

function factionShort(faction) {
  const value = String(faction || '').toUpperCase();
  if (value === 'RESISTANCE') return 'RES';
  if (value === 'ENLIGHTENED') return 'ENL';
  return 'NEU';
}

function formatNumber(value) {
  return Number(value || 0).toLocaleString('en-US');
}

export default function TopBar({ onOpen, player, ap }) {
  const username = player?.username || 'Agent';
  const level = Number(player?.level || 1);
  const xm = Number(player?.xm || 0);
  const maxXm = Math.max(1, Number(player?.maxXm || 1));
  const xmPercent = Math.max(0, Math.min(100, (xm / maxXm) * 100));

  return (
    <header className="top-bar">
      <button className="player-chip player-chip-btn" onClick={() => onOpen('player')}>
        <div className="player-avatar">{username.slice(0, 1).toUpperCase()}</div>
        <div className="player-info">
          <span className="player-name">{username}</span>
          <span className="player-level">L{level} · {factionShort(player?.faction)}</span>

          <div className="xm-bar" role="img" aria-label={`XM ${xm} / ${maxXm}`}>
            <div className="xm-bar-track">
              <div className="xm-bar-fill" style={{ width: `${xmPercent}%` }} />
            </div>
            <div className="xm-bar-text">
              <span>{formatNumber(xm)}</span>
              <small>/ {formatNumber(maxXm)} XM</small>
            </div>
          </div>

          <div className="ap-mini" role="img" aria-label={`AP ${ap?.text || ''}`}>
            <div className="ap-mini-track">
              <div className="ap-mini-fill" style={{ width: `${ap?.percent || 0}%` }} />
            </div>
            <div className="ap-mini-text">{ap?.text || `${formatNumber(player?.ap)} AP`}</div>
          </div>
        </div>
      </button>

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
