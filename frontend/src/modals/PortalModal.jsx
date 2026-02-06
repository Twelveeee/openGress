import React from 'react';
import Modal from '../components/Modal.jsx';
import { getInstalledModIcon } from '../utils/itemIcons.js';

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

export default function PortalModal({ open, onClose, portal, onDeploy, onModDeploy, onLink, onCharge }) {
  const name = portal?.name || 'Portal';
  const level = portal?.level ?? 1;
  const faction = portal?.faction || 'NEUTRAL';
  const image = portal?.image || '';
  const owner = portal?.owner || '—';
  const description = portal?.description || `Faction: ${factionLabel(faction)}`;
  const resonators = portal?.resonators || Array.from({ length: 8 }, () => null);
  const leftRes = resonators.slice(0, 4);
  const rightRes = resonators.slice(4);
  const ownerClass = factionClass(faction);
  const portalLevelClass = levelClass(level);
  const actions = [
    { key: 'deploy', label: 'Deploy', icon: '◐', onClick: onDeploy },
    { key: 'link', label: 'Link', icon: '↗', onClick: onLink },
    { key: 'charge', label: 'Charge', icon: '⚡', onClick: onCharge }
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
              <span>· {factionLabel(faction)}</span>
              <span>
                · Owner: <span className={ownerClass}>{owner}</span>
              </span>
            </div>
          </div>
        </div>
        <div className="portal-right">
          <div className="portal-photo" style={image ? { backgroundImage: `url(${image})` } : undefined} />
          <div className="mod-grid">
            {(portal?.mods || Array.from({ length: 4 }, () => null)).map((slot, idx) => (
              <button key={idx} className={`mod-slot ${slot?.subtype ? '' : 'empty'}`} onClick={onModDeploy}>
                <img className="mod-slot-icon" src={getInstalledModIcon(slot)} alt={slot?.type || 'empty'} />
              </button>
            ))}
          </div>
        </div>
      </header>
      <div className="portal-core-area">
        <div className="res-column left">
          {leftRes.map((res, idx) => (
            <div key={`${res?.owner || 'empty'}-${idx}`} className="res-row left">
              <div className={`res-xm ${factionClass(res?.faction || faction)}`}>
                <span className="res-bar">
                  <span className="res-bar-fill" style={{ height: `${res?.xm ?? 0}%` }} />
                  {(res?.xm ?? 0) >= 100 ? <span className="res-bar-cap" /> : null}
                </span>
              </div>
              <div className="res-text">
                {res?.level ? (
                  <span className={`res-level ${levelClass(res.level)}`}>L{res.level}</span>
                ) : (
                  <span className="res-level">—</span>
                )}
                <span className={`res-owner ${factionClass(res?.faction || faction)}`}>{res?.owner || ''}</span>
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
            <div key={`${res?.owner || 'empty'}-${idx}`} className="res-row right">
              <div className="res-text">
                {res?.level ? (
                  <span className={`res-level ${levelClass(res.level)}`}>L{res.level}</span>
                ) : (
                  <span className="res-level">—</span>
                )}
                <span className={`res-owner ${factionClass(res?.faction || faction)}`}>{res?.owner || ''}</span>
              </div>
              <div className={`res-xm ${factionClass(res?.faction || faction)}`}>
                <span className="res-bar">
                  <span className="res-bar-fill" style={{ height: `${res?.xm ?? 0}%` }} />
                  {(res?.xm ?? 0) >= 100 ? <span className="res-bar-cap" /> : null}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
      <footer className="portal-actions portal-actions-3">
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
