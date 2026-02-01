import React, { useMemo, useRef, useState } from 'react';
import Modal from '../components/Modal.jsx';
import ItemUseBar from '../components/ItemUseBar.jsx';

function levelClass(level) {
  if (level <= 3) return 'lvl-low';
  if (level <= 6) return 'lvl-mid';
  return 'lvl-high';
}

export default function PortalModDeployModal({ open, onClose, portal, items = [], playerName = 'Agent' }) {
  const mods = useMemo(
    () => items.filter((item) => item.type === 'mod' && item.count > 0),
    [items]
  );
  const sortedMods = useMemo(() => mods.slice(), [mods]);
  const [selectedId, setSelectedId] = useState(sortedMods[0]?.id || null);
  const selected = sortedMods.find((item) => item.id === selectedId) || sortedMods[0];

  const [slots, setSlots] = useState(() =>
    Array.from({ length: 4 }, (_, idx) => ({
      id: idx,
      type: null,
      owner: null,
      rarity: null,
      subtype: null
    }))
  );

  const [actionNotice, setActionNotice] = useState('');
  const [actionNoticeKey, setActionNoticeKey] = useState(0);
  const [actionError, setActionError] = useState(false);
  const noticeTimerRef = useRef(null);
  const [selectedSlot, setSelectedSlot] = useState(0);

  const name = portal?.name || 'Portal';
  const level = portal?.level ?? 1;
  const portalFaction = portal?.faction || 'RESISTANCE';

  const showNotice = (message) => {
    if (noticeTimerRef.current) clearTimeout(noticeTimerRef.current);
    setActionNotice(message);
    setActionNoticeKey((prev) => prev + 1);
    setActionError(true);
    noticeTimerRef.current = setTimeout(() => {
      setActionNotice('');
      setActionError(false);
    }, 200);
  };

  const playerModCount = slots.filter((slot) => slot.owner === playerName).length;
  const nextEmptySlot = (current) => {
    for (let i = 1; i <= 4; i += 1) {
      const next = (current + i) % 4;
      if (!slots[next].type) return next;
    }
    return current;
  };

  const handleInstall = () => {
    if (!selected) return;
    if (slots[selectedSlot].type) {
      showNotice('slot occupied');
      return;
    }
    if (playerModCount >= 2 && slots[selectedSlot].owner !== playerName) {
      showNotice('mod limit reached');
      return;
    }
    setSlots((prev) =>
      prev.map((slot, idx) =>
        idx === selectedSlot
          ? {
              ...slot,
              type: selected.name,
              owner: playerName,
              rarity: selected.rarity || 'C',
              subtype: selected.subtype || 'SHIELD'
            }
          : slot
      )
    );
    setSelectedSlot(nextEmptySlot(selectedSlot));
  };

  const modEffect = (mod) => {
    if (!mod) return null;
    const rarity = mod.rarity || 'C';
    const rarityValue = { C: 10, R: 15, VR: 20 };
    switch (mod.subtype) {
      case 'SHIELD':
        return { label: 'Shielding', value: rarityValue[rarity] || 0 };
      case 'LINK':
        return { label: 'Link Range', value: rarity === 'VR' ? 7 : 2 };
      case 'FORCE':
        return { label: 'Force Amp', value: rarity === 'VR' ? 30 : rarity === 'R' ? 25 : 20 };
      case 'TURRET':
        return { label: 'Turret', value: rarity === 'VR' ? 30 : rarity === 'R' ? 20 : 10 };
      case 'HEATSINK':
        return { label: 'Cooldown', value: rarity === 'VR' ? -70 : rarity === 'R' ? -50 : -20 };
      case 'MULTI':
        return { label: 'XM Spin', value: rarity === 'VR' ? 8 : rarity === 'R' ? 6 : 4 };
      default:
        return null;
    }
  };

  const installedEffects = slots
    .map((slot) => (slot.subtype ? modEffect(slot) : null))
    .filter(Boolean)
    .reduce((acc, effect) => {
      acc[effect.label] = (acc[effect.label] || 0) + effect.value;
      return acc;
    }, {});

  const selectedEffect = modEffect(selected);

  return (
    <Modal open={open} onClose={onClose} className="portal-deploy-card">
      <header className="modal-header portal-header">
        <div className="portal-deploy-box">
          <div className="portal-deploy-title">
            <span className={`portal-level ${levelClass(level)}`}>L{level}</span>
            <span className="portal-deploy-name">{name}</span>
          </div>
          {Object.entries(installedEffects).map(([label, value]) => {
            const isSame = selectedEffect && selectedEffect.label === label;
            return (
              <div key={label} className="portal-deploy-line">
                <span>{label}</span>
                <span className="portal-deploy-delta">
                  {value}
                  {isSame ? ` [+${Math.abs(selectedEffect.value)}]` : ''}
                </span>
              </div>
            );
          })}
          {selectedEffect && !installedEffects[selectedEffect.label] ? (
            <div className="portal-deploy-line">
              <span>{selectedEffect.label}</span>
              <span className="portal-deploy-delta">[+{Math.abs(selectedEffect.value)}]</span>
            </div>
          ) : null}
        </div>
      </header>

      <div className="portal-mod-body">
        <div className="mod-slot-grid">
          {slots.map((slot, idx) => (
            <div key={slot.id} className={`mod-slot-wrap ${idx < 2 ? 'top' : 'bottom'}`}>
              <span className={`mod-slot-label ${idx < 2 ? 'top' : 'bottom'}`}>
                {idx + 1}{' '}
                <span className={portalFaction === 'RESISTANCE' ? 'res' : 'enl'}>
                  {playerName}
                </span>
              </span>
              <button
                className={`mod-slot-cell ${selectedSlot === idx ? 'active' : ''}`}
                onClick={() => setSelectedSlot(idx)}
              >
                {slot.type ? (
                  <span className={`rarity ${slot.rarity || 'C'}`}>///</span>
                ) : (
                  <span className="mod-slot-empty" />
                )}
              </button>
            </div>
          ))}
        </div>
      </div>

      <ItemUseBar
        helpText="Install mod"
        actionLabel="INSTALL"
        items={sortedMods}
        selectedId={selected?.id}
        onSelect={setSelectedId}
        onAction={handleInstall}
        notice={actionNotice}
        noticeKey={actionNoticeKey}
        actionClassName={actionError ? 'error' : ''}
        itemMode="rarity"
        footerText={selected ? `Install Mod: ${selected.name}` : 'Install Mod'}
      />
    </Modal>
  );
}
