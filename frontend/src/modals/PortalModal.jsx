import React from 'react';
import Modal from '../components/Modal.jsx';

function factionLabel(faction) {
  switch (faction) {
    case 'RESISTANCE':
      return 'Resistance';
    case 'ENLIGHTENED':
      return 'Enlightened';
    case 'NEUTRAL':
      return 'Neutral';
    default:
      return 'Unknown';
  }
}

function factionClass(faction) {
  switch (faction) {
    case 'RESISTANCE':
      return 'res';
    case 'ENLIGHTENED':
      return 'enl';
    case 'NEUTRAL':
      return 'neutral';
    default:
      return 'red';
  }
}

function levelClass(level) {
  if (level <= 3) return 'lvl-low';
  if (level <= 6) return 'lvl-mid';
  return 'lvl-high';
}

export default function PortalModal({ open, onClose, portal, onDeploy, onModDeploy }) {
  const name = portal?.name || 'Arcoiris';
  const level = portal?.level ?? 7;
  const faction = portal?.faction || 'RESISTANCE';
  const image = portal?.image || '';
  const owner = portal?.owner || 'catislife';
  const distance = portal?.distance ?? 13900;
  const description =
    portal?.description ||
    'This was meant to be a rainbow, but the installation of Arcoiris, there is a...';
  const resonators =
    portal?.resonators ||
    [
      { level: 1, owner: 'GPC2C', xm: 100, faction: 'RESISTANCE' },
      { level: 2, owner: 'miester', xm: 92, faction: 'ENLIGHTENED' },
      { level: 3, owner: 'RogerDodg.', xm: 84, faction: 'RESISTANCE' },
      { level: 4, owner: 'speakerTo...', xm: 76, faction: 'ENLIGHTENED' },
      { level: 5, owner: 'XderIon', xm: 68, faction: 'RESISTANCE' },
      { level: 6, owner: 'ParrotCay', xm: 60, faction: 'ENLIGHTENED' },
      { level: 7, owner: 'RogerDodg.', xm: 52, faction: 'RESISTANCE' },
      { level: 8, owner: 'catislife', xm: 100, faction: 'ENLIGHTENED' }
    ];
  const leftRes = resonators.slice(0, 4);
  const rightRes = resonators.slice(4);
  const ownerClass = factionClass(faction);
  const portalLevelClass = levelClass(level);
  const actions = [
    { key: 'deploy', label: 'Deploy', icon: '◐', onClick: onDeploy },
    { key: 'link', label: 'Link', icon: '↗' },
    { key: 'hack', label: 'Hack', icon: '◎' },
    { key: 'charge', label: 'Charge', icon: '⚡' }
  ];

  return (
    <Modal open={open} onClose={onClose} className="portal-card">
      <header className="modal-header portal-header">
        <div className="portal-title">
          <span className={`portal-level ${portalLevelClass}`}>{level}</span>
          <div>
            <h3>{name}</h3>
            <p className="muted">{description}</p>
            <div className="portal-meta">
              <span>· {Math.round(distance / 100) / 10}km</span>
              <span>
                · Owner: <span className={ownerClass}>{owner}</span>
              </span>
            </div>
          </div>
        </div>
        <div className="portal-right">
        <div
          className="portal-photo"
          style={image ? { backgroundImage: `url(${image})` } : undefined}
        />
        <div className="mod-grid">
          <button className="mod-slot" onClick={onModDeploy}>⌁</button>
          <button className="mod-slot" onClick={onModDeploy}>⌁</button>
          <button className="mod-slot" onClick={onModDeploy}>⌁</button>
          <button className="mod-slot empty" onClick={onModDeploy}>+</button>
        </div>
        </div>
      </header>
      <div className="portal-core-area">
        <div className="res-column left">
          {leftRes.map((res, idx) => (
            <div key={`${res.owner}-${idx}`} className="res-row left">
              <div className={`res-xm ${factionClass(res.faction || faction)}`}>
                <span className="res-bar">
                  <span className="res-bar-fill" style={{ height: `${res.xm ?? 100}%` }} />
                  {(res.xm ?? 100) >= 100 ? <span className="res-bar-cap" /> : null}
                </span>
              </div>
              <div className="res-text">
                <span className={`res-level ${levelClass(res.level)}`}>L{res.level}</span>
                <span className={`res-owner ${factionClass(res.faction || faction)}`}>
                  {res.owner}
                </span>
              </div>
            </div>
          ))}
        </div>
        <div className="portal-core">
          <div className="portal-ring" />
          <div className="portal-node" />
        </div>
        <div className="res-column right">
          {rightRes.map((res, idx) => (
            <div key={`${res.owner}-${idx}`} className="res-row right">
              <div className="res-text">
                <span className={`res-level ${levelClass(res.level)}`}>L{res.level}</span>
                <span className={`res-owner ${factionClass(res.faction || faction)}`}>
                  {res.owner}
                </span>
              </div>
              <div className={`res-xm ${factionClass(res.faction || faction)}`}>
                <span className="res-bar">
                  <span className="res-bar-fill" style={{ height: `${res.xm ?? 100}%` }} />
                  {(res.xm ?? 100) >= 100 ? <span className="res-bar-cap" /> : null}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
      <footer className="portal-actions">
        {actions.map((action) => (
          <button key={action.key} className="portal-action" onClick={action.onClick}>
            <span className="portal-action-icon">{action.icon}</span>
            <span>{action.label}</span>
          </button>
        ))}
      </footer>
    </Modal>
  );
}
