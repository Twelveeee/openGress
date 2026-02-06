import React, { useEffect, useMemo, useRef, useState } from 'react';
import Modal from '../components/Modal.jsx';
import ItemUseBar from '../components/ItemUseBar.jsx';

function sortWeapons(a, b) {
  const order = { XMP: 1, US: 2 };
  const aType = order[String(a.subtype || '').toUpperCase()] || 99;
  const bType = order[String(b.subtype || '').toUpperCase()] || 99;
  if (aType !== bType) return aType - bType;
  return Number(a.level || 0) - Number(b.level || 0);
}

export default function AttackModal({ open, onClose, items = [], onAttack, previewTargets }) {
  const weapons = useMemo(
    () =>
      items
        .filter((item) => item.type === 'weapon' && ['XMP', 'US'].includes(String(item.subtype || '').toUpperCase()))
        .sort(sortWeapons),
    [items]
  );

  const [selectedId, setSelectedId] = useState(weapons[0]?.id || null);
  const [charging, setCharging] = useState(false);
  const [charge, setCharge] = useState(0);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const chargeRef = useRef(null);

  const selected = weapons.find((item) => item.id === selectedId) || weapons[0];
  const targets = useMemo(() => (selected ? previewTargets?.(selected) || [] : []), [previewTargets, selected]);

  useEffect(() => {
    if (!open) {
      setCharging(false);
      setCharge(0);
      setBusy(false);
      setNotice('');
      return;
    }
    if (!selected && weapons[0]) {
      setSelectedId(weapons[0].id);
    }
  }, [open, selected, weapons]);

  useEffect(() => {
    if (!charging) return undefined;
    const start = performance.now();

    const tick = (now) => {
      const t = Math.min((now - start) / 1200, 1);
      setCharge(t);
      chargeRef.current = requestAnimationFrame(tick);
    };

    chargeRef.current = requestAnimationFrame(tick);
    return () => {
      if (chargeRef.current) cancelAnimationFrame(chargeRef.current);
    };
  }, [charging]);

  const fire = async () => {
    if (!selected || busy) return;
    setBusy(true);
    const bonus = charge >= 0.7 ? 0.2 : charge >= 0.35 ? 0.1 : 0;
    try {
      const result = await onAttack?.({ item: selected, charge: bonus });
      if (result?.ok) {
        setNotice(`命中 ${result.count || 0} 个目标`);
      } else if (result?.count === 0) {
        setNotice('范围内没有目标');
      }
    } finally {
      setBusy(false);
      setCharge(0);
    }
  };

  const handleFireClick = () => {
    if (charging || busy) return;
    fire();
  };

  const handleChargeStart = () => {
    if (!selected || busy) return;
    setCharging(true);
  };

  const handleChargeEnd = () => {
    if (!charging) return;
    setCharging(false);
    fire();
  };

  useEffect(() => {
    const handleKeyDown = (event) => {
      if (!open) return;
      if (event.code === 'Space' && !event.repeat) {
        event.preventDefault();
        handleChargeStart();
      }
    };

    const handleKeyUp = (event) => {
      if (!open) return;
      if (event.code === 'Space') {
        event.preventDefault();
        handleChargeEnd();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
    };
  }, [open, charging, busy, selected]);

  return (
    <Modal open={open} onClose={onClose} className="attack-card" placement="bottom">
      <header className="modal-header">
        <h3>Attack</h3>
        <span className="muted">范围目标: {targets.length}</span>
      </header>
      <div className="attack-body">
        <ItemUseBar
          helpText="Fire weapon"
          actionLabel={busy ? 'FIRING...' : 'FIRE'}
          items={weapons}
          selectedId={selected?.id}
          onSelect={setSelectedId}
          onAction={handleFireClick}
          actionClassName={charging ? 'charging' : ''}
          onActionMouseDown={handleChargeStart}
          onActionMouseUp={handleChargeEnd}
          onActionMouseLeave={handleChargeEnd}
          onActionTouchStart={handleChargeStart}
          onActionTouchEnd={handleChargeEnd}
          actionDisabled={!selected || busy}
          notice={notice}
          noticeKey={`${selected?.id || 'none'}-${targets.length}`}
          footerText={
            selected
              ? `${selected.name} L${selected.level || 1} · in-range ${targets.length}`
              : '—'
          }
        />
      </div>
    </Modal>
  );
}
