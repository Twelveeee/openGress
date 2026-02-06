import React, { useMemo, useRef, useState } from 'react';
import Modal from '../components/Modal.jsx';
import ItemUseBar from '../components/ItemUseBar.jsx';
import { getInstalledModIcon } from '../utils/itemIcons.js';

function levelClass(level) {
  if (level <= 3) return 'lvl-low';
  if (level <= 6) return 'lvl-mid';
  return 'lvl-high';
}

export default function PortalModDeployModal({
  open,
  onClose,
  portal,
  modSlots = [],
  onUpdateSlots,
  items = [],
  playerName = 'Agent',
  playerFaction = 'RESISTANCE'
}) {
  const formatValue = (value) => (Number.isInteger(value) ? value : value.toFixed(2));
  const mods = useMemo(
    () => items.filter((item) => item.type === 'mod' && item.count > 0),
    [items]
  );
  const sortedMods = useMemo(() => mods.slice(), [mods]);
  const [selectedId, setSelectedId] = useState(sortedMods[0]?.id || null);
  const selected = sortedMods.find((item) => item.id === selectedId) || sortedMods[0];

  const resolvedSlots = Array.from({ length: 4 }, (_, idx) => {
    const slot = modSlots[idx];
    if (slot) {
      return { id: idx, ...slot };
    }
    return {
      id: idx,
      type: null,
      owner: null,
      rarity: null,
      subtype: null
    };
  });

  const [actionNotice, setActionNotice] = useState('');
  const [actionNoticeKey, setActionNoticeKey] = useState(0);
  const [actionError, setActionError] = useState(false);
  const noticeTimerRef = useRef(null);
  const [selectedSlot, setSelectedSlot] = useState(0);

  const name = portal?.name || 'Portal';
  const level = portal?.level ?? 1;
  const portalFaction = portal?.faction || 'RESISTANCE';
  const portalFactionClass =
    portalFaction === 'RESISTANCE' ? 'res' : portalFaction === 'ENLIGHTENED' ? 'enl' : 'neutral';

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

  const playerModCount = resolvedSlots.filter((slot) => slot.owner === playerName).length;
  const nextEmptySlot = (current) => {
    for (let i = 1; i <= 4; i += 1) {
      const next = (current + i) % 4;
      if (!resolvedSlots[next].type) return next;
    }
    return current;
  };

  const canInstall =
    portal?.faction &&
    portal?.faction !== 'NEUTRAL' &&
    portalFaction === String(playerFaction || '').toUpperCase();

  const handleInstall = () => {
    if (!canInstall) {
      showNotice('先占领');
      return;
    }
    if (!selected) return;
    if (resolvedSlots[selectedSlot].type) {
      showNotice('slot occupied');
      return;
    }
    if (playerModCount >= 2 && resolvedSlots[selectedSlot].owner !== playerName) {
      showNotice('mod limit reached');
      return;
    }
    const nextSlots = resolvedSlots.map((slot, idx) => {
      if (idx === selectedSlot) {
        return {
          ...slot,
          type: selected.name,
          owner: playerName,
          rarity: selected.rarity || 'C',
          subtype: selected.subtype || 'SHIELD',
          modType: selected.modType || selected.subtype || 'SHIELD'
        };
      }
      return slot?.subtype ? slot : null;
    });
    onUpdateSlots?.(portal?.id, nextSlots);
    setSelectedSlot(nextEmptySlot(selectedSlot));
  };

  const modEffect = (mod) => {
    if (!mod) return [];
    const rarity = mod.rarity || 'C';
    const shieldValues = { C: 30, R: 40, VR: 60, AXA: 70 };
    const multiValues = { C: 4, R: 8, VR: 12 };
    switch (mod.subtype) {
      case 'SHIELD':
        return [{ label: 'Shielding', value: shieldValues[rarity] || 0 }];
      case 'LINK':
        return [{ label: 'Link Range', value: rarity === 'VR' ? 7 : 2 }];
      case 'SBUL':
        return [{ label: 'Link Range', value: 5 }];
      case 'FORCE':
        return [{ label: 'Force Amp', value: 2.0 }];
      case 'TURRET':
        return [
          { label: 'Turret Rate', value: 2.0 },
          { label: 'Turret Crit', value: 30 }
        ];
      case 'HEATSINK':
        return [{ label: 'Cooldown', value: rarity === 'VR' ? -70 : rarity === 'R' ? -50 : -20 }];
      case 'MULTI':
        return [{ label: 'XM Spin', value: multiValues[rarity] || 0 }];
      default:
        return [];
    }
  };

  const installedEffects = resolvedSlots
    .flatMap((slot) => (slot.subtype ? modEffect(slot) : []))
    .filter(Boolean)
    .reduce((acc, effect) => {
      acc[effect.label] = (acc[effect.label] || 0) + effect.value;
      return acc;
    }, {});

  const selectedEffects = (modEffect(selected) || []).filter(Boolean);
  const selectedEffectsMap = selectedEffects.reduce((acc, effect) => {
    acc[effect.label] = (acc[effect.label] || 0) + effect.value;
    return acc;
  }, {});

  return (
    <Modal open={open} onClose={onClose} className="portal-deploy-card portal-mod-card">
      <header className="modal-header portal-header">
        <div className="portal-deploy-box">
          <div className="portal-deploy-title">
            <span className={`portal-level ${levelClass(level)}`}>L{level}</span>
            <span className="portal-deploy-name">{name}</span>
          </div>
          {Object.entries(installedEffects).map(([label, value]) => {
            const delta = selectedEffectsMap[label];
            const deltaText =
              delta != null
                ? ` [${delta >= 0 ? '+' : ''}${formatValue(delta)}]`
                : '';
            return (
              <div key={label} className="portal-deploy-line">
                <span>{label}</span>
                <span className="portal-deploy-delta">
                  {formatValue(value)}
                  {deltaText}
                </span>
              </div>
            );
          })}
          {selectedEffects
            .filter((effect) => installedEffects[effect.label] == null)
            .map((effect) => (
              <div key={effect.label} className="portal-deploy-line">
                <span>{effect.label}</span>
                <span className="portal-deploy-delta">
                  [{effect.value >= 0 ? '+' : ''}{formatValue(effect.value)}]
                </span>
              </div>
            ))}
        </div>
      </header>

      <div className="portal-mod-body">
        <div className="mod-slot-grid">
          {resolvedSlots.map((slot, idx) => (
            <div key={slot.id} className={`mod-slot-wrap ${idx < 2 ? 'top' : 'bottom'}`}>
              <span className={`mod-slot-label ${idx < 2 ? 'top' : 'bottom'}`}>
                {idx + 1}{' '}
                <span className={portalFactionClass}>{playerName}</span>
              </span>
              <button
                className={`mod-slot-cell ${selectedSlot === idx ? 'active' : ''}`}
                onClick={() => setSelectedSlot(idx)}
              >
                <img
                  className="mod-slot-installed-icon"
                  src={getInstalledModIcon(slot)}
                  alt={slot?.type || 'empty'}
                />
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
        actionDisabled={!canInstall}
        actionDisabledText="先占领"
      />
    </Modal>
  );
}
