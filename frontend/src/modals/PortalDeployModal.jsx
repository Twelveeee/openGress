import React, { useEffect, useMemo, useRef, useState } from 'react';
import Modal from '../components/Modal.jsx';
import ItemUseBar from '../components/ItemUseBar.jsx';

function levelClass(level) {
  if (level <= 3) return 'lvl-low';
  if (level <= 6) return 'lvl-mid';
  return 'lvl-high';
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

export default function PortalDeployModal({
  open,
  onClose,
  portal,
  slots = [],
  onUpdateSlots,
  items = [],
  playerName = 'Agent'
}) {
  const resonators = useMemo(
    () => items.filter((item) => item.type === 'resonator' && item.count > 0),
    [items]
  );
  const [selectedSlot, setSelectedSlot] = useState(0);
  const [selectedId, setSelectedId] = useState(resonators[0]?.id || null);
  const [lastSelectedLevel, setLastSelectedLevel] = useState(null);
  const [actionNotice, setActionNotice] = useState('');
  const [actionNoticeKey, setActionNoticeKey] = useState(0);
  const [actionError, setActionError] = useState(false);
  const noticeTimerRef = useRef(null);

  const resolvedSlots = Array.from({ length: 8 }, (_, idx) => {
    const slot = slots[idx];
    if (slot) {
      return { id: idx, ...slot };
    }
    return {
      id: idx,
      level: null,
      owner: null,
      faction: null,
      xm: 0
    };
  });
  const [slotFx, setSlotFx] = useState({});

  const name = portal?.name || 'Portal';
  const portalFaction = portal?.faction || 'RESISTANCE';

  const xmByLevel = {
    1: 1000,
    2: 1500,
    3: 2000,
    4: 2500,
    5: 3000,
    6: 4000,
    7: 5000,
    8: 6000
  };

  const computeStats = (nextSlots) => {
    const total = nextSlots.reduce((sum, slot) => sum + (slot.level || 0), 0);
    if (total === 0) {
      return { level: 1, range: 0, energy: 0, energyMax: 0 };
    }
    const avg = total / 8;
    const level = Math.max(1, Math.floor(avg));
    const range = avg > 0 ? 160 * Math.pow(avg, 4) : 0;
    const energyMax = nextSlots.reduce(
      (sum, slot) => sum + (xmByLevel[slot.level] || 0),
      0
    );
    const energy = Math.floor(energyMax * 0.7);
    return { level, range, energy, energyMax };
  };

  const currentStats = computeStats(resolvedSlots);

  const order = [0, 1, 2, 3, 4, 5, 6, 7];
  const nextEmptySlot = (current) => {
    const startIdx = order.indexOf(current);
    for (let i = 1; i <= order.length; i += 1) {
      const nextId = order[(startIdx + i) % order.length];
      if (!resolvedSlots[nextId]?.level) return nextId;
    }
    return current;
  };

  const limitsByLevel = {
    8: 1,
    7: 1,
    6: 2,
    5: 2,
    4: 4,
    3: 4,
    2: 4,
    1: 8
  };

  const countsByLevel = resolvedSlots.reduce((acc, slot) => {
    if (slot.owner !== playerName || !slot.level) return acc;
    acc[slot.level] = (acc[slot.level] || 0) + 1;
    return acc;
  }, {});

  const canUseLevel = (level, slotId) => {
    const limit = limitsByLevel[level] ?? 0;
    const current = countsByLevel[level] || 0;
    const slot = resolvedSlots[slotId];
    if (slot && slot.owner === playerName && slot.level === level) return true;
    if (slot && slot.owner === playerName && slot.level) {
      const adjusted = level === slot.level ? current : current;
      return adjusted < limit;
    }
    return current < limit;
  };

  const allResonators = useMemo(
    () => resonators.slice().sort((a, b) => (a.level || 0) - (b.level || 0)),
    [resonators]
  );

  const selected =
    allResonators.find((item) => item.id === selectedId) || allResonators[allResonators.length - 1];

  const withSelected = (() => {
    if (!selected) return currentStats;
    const nextSlots = resolvedSlots.map((slot, idx) =>
      idx === selectedSlot ? { ...slot, level: selected.level, owner: playerName } : slot
    );
    return computeStats(nextSlots);
  })();

  const deltaLevel = withSelected.level - currentStats.level;
  const deltaRange = withSelected.range - currentStats.range;

  const actionLabel = resolvedSlots[selectedSlot]?.level ? 'UPGRADE' : 'DEPLOY';

  const preferredId = useMemo(() => {
    if (!allResonators.length) return null;
    const sorted = allResonators.slice();
    const targetLevel = lastSelectedLevel || sorted[sorted.length - 1]?.level || 0;
    const nextLowerOrEqual = [...sorted]
      .reverse()
      .find((item) => (item.level || 0) <= targetLevel);
    const fallback = sorted[sorted.length - 1];
    return (nextLowerOrEqual || fallback)?.id || null;
  }, [allResonators, lastSelectedLevel]);

  useEffect(() => {
    if (!preferredId) return;
    if (selectedId !== preferredId) {
      setSelectedId(preferredId);
    }
  }, [preferredId, selectedId]);

  useEffect(() => {
    if (!selected?.level) return;
    if (lastSelectedLevel == null) {
      setLastSelectedLevel(selected.level);
    }
  }, [selected, lastSelectedLevel]);

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

  const handleDeploy = () => {
    if (!selected) return;
    const slot = resolvedSlots[selectedSlot];
    if (slot.level && (selected.level || 0) <= slot.level) {
      showNotice('not an upgrade');
      return;
    }
    const limit = limitsByLevel[selected.level || 0] ?? 0;
    const current = countsByLevel[selected.level || 0] || 0;
    const sameLevelInSlot =
      slot.owner === playerName && slot.level === (selected.level || 0);
    const projected = sameLevelInSlot ? current : current + 1;
    if (projected > limit) {
      showNotice('level limit reached');
      return;
    }
    const wasOccupied = Boolean(resolvedSlots[selectedSlot]?.level);
    setLastSelectedLevel(selected.level || null);
    const level = selected.level || 0;
    const limitCount = limitsByLevel[level] ?? 0;
    const currentCount = countsByLevel[level] || 0;
    const prevLevel =
      slot.owner === playerName && slot.level ? slot.level : null;
    const nextCount =
      currentCount + (prevLevel === level ? 0 : 1) - (prevLevel && prevLevel !== level ? 1 : 0);
    const levelsDesc = allResonators
      .map((item) => item.level || 0)
      .filter((val) => val > 0)
      .sort((a, b) => b - a);
    const nextLowerLevel = levelsDesc.find((val) => val < level) || level;
    const nextSlots = resolvedSlots.map((slot, idx) =>
      idx === selectedSlot
        ? {
            ...slot,
            level: selected.level,
            owner: playerName,
            faction: portalFaction,
            xm: 100
          }
        : slot
    );
    onUpdateSlots?.(portal?.id, nextSlots);
    setSlotFx((prev) => ({ ...prev, [selectedSlot]: wasOccupied ? 'upgrade' : 'deploy' }));
    setTimeout(() => {
      setSlotFx((prev) => {
        const next = { ...prev };
        delete next[selectedSlot];
        return next;
      });
    }, 400);
    if (nextCount >= limitCount) {
      setLastSelectedLevel(nextLowerLevel);
    }
    if (!wasOccupied) {
      setSelectedSlot(nextEmptySlot(selectedSlot));
    }
  };

  return (
    <Modal open={open} onClose={onClose} className="portal-deploy-card">
      <header className="modal-header portal-header">
        <div className="portal-deploy-box">
          <div className="portal-deploy-title">
            <span className={`portal-level ${levelClass(currentStats.level)}`}>
              L{currentStats.level}
            </span>
            <span className="portal-deploy-delta">
              [{deltaLevel >= 0 ? '+' : ''}{deltaLevel}]
            </span>
            <span className="portal-deploy-name">{name}</span>
          </div>
          <div className="portal-deploy-line">
            <span>Range: {currentStats.range.toFixed(1)}m</span>
            <span className="portal-deploy-delta">
              [{deltaRange >= 0 ? '+' : ''}{deltaRange.toFixed(0)} m]
            </span>
          </div>
          <div className="portal-deploy-line">
            <span>
              Energy: {currentStats.energy} / {currentStats.energyMax}
            </span>
          </div>
        </div>
      </header>

      <div className="portal-deploy-body">
        <div className="deploy-side left">
          {resolvedSlots.slice(0, 4).map((slot) => (
            <button
              key={slot.id}
              className={`deploy-slot left ${selectedSlot === slot.id ? 'active' : ''} ${
                slotFx[slot.id] ? `fx-${slotFx[slot.id]}` : ''
              }`}
              onClick={() => setSelectedSlot(slot.id)}
            >
              <div
                className={`res-xm ${factionClass(slot.faction || portalFaction || 'NEUTRAL')} ${
                  selectedSlot === slot.id ? 'active' : ''
                }`}
              >
                <span className="res-bar">
                  <span className="res-bar-fill" style={{ height: `${slot.xm || 0}%` }} />
                  {slot.xm >= 100 ? <span className="res-bar-cap" /> : null}
                </span>
              </div>
              <div className="res-text">
                {slot.level ? (
                  <span className={`res-level ${levelClass(slot.level)}`}>L{slot.level}</span>
                ) : (
                  <span className="res-level">—</span>
                )}
                <span className={`res-owner ${factionClass(slot.faction || portalFaction || 'NEUTRAL')}`}>
                  {slot.owner || ''}
                </span>
              </div>
            </button>
          ))}
        </div>
        <div className="deploy-core">
          <div className="portal-ring" />
          <div className="portal-node" />
        </div>
        <div className="deploy-side right">
          {resolvedSlots.slice(4).map((slot) => (
            <button
              key={slot.id}
              className={`deploy-slot right ${selectedSlot === slot.id ? 'active' : ''} ${
                slotFx[slot.id] ? `fx-${slotFx[slot.id]}` : ''
              }`}
              onClick={() => setSelectedSlot(slot.id)}
            >
              <div className="res-text">
                {slot.level ? (
                  <span className={`res-level ${levelClass(slot.level)}`}>L{slot.level}</span>
                ) : (
                  <span className="res-level">—</span>
                )}
                <span className={`res-owner ${factionClass(slot.faction || portalFaction || 'NEUTRAL')}`}>
                  {slot.owner || ''}
                </span>
              </div>
              <div
                className={`res-xm ${factionClass(slot.faction || portalFaction || 'NEUTRAL')} ${
                  selectedSlot === slot.id ? 'active' : ''
                }`}
              >
                <span className="res-bar">
                  <span className="res-bar-fill" style={{ height: `${slot.xm || 0}%` }} />
                  {slot.xm >= 100 ? <span className="res-bar-cap" /> : null}
                </span>
              </div>
            </button>
          ))}
        </div>
      </div>

      <ItemUseBar
        helpText="Deploy resonator"
        actionLabel={actionLabel}
        items={allResonators}
        selectedId={selected?.id}
        onSelect={(id) => {
          const item = allResonators.find((entry) => entry.id === id);
          setSelectedId(id);
          if (item?.level) setLastSelectedLevel(item.level);
        }}
        onAction={handleDeploy}
        notice={actionNotice}
        noticeKey={actionNoticeKey}
        actionClassName={actionError ? 'error' : ''}
        footerText="Deploy Resonator"
      />
    </Modal>
  );
}
