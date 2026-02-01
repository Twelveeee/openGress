import React, { useEffect, useMemo, useRef, useState } from 'react';
import Modal from '../components/Modal.jsx';
import ItemUseBar from '../components/ItemUseBar.jsx';

const typeOrder = ['CUBE', 'XMP', 'US', 'VIRUS'];

function sortWeapons(a, b) {
  const typeDiff = typeOrder.indexOf(a.subtype) - typeOrder.indexOf(b.subtype);
  if (typeDiff !== 0) return typeDiff;
  return (a.level || 0) - (b.level || 0);
}

export default function AttackModal({ open, onClose, items = [], onConsume }) {
  const weapons = useMemo(
    () => items.filter((item) => ['weapon', 'cube'].includes(item.type)).sort(sortWeapons),
    [items]
  );

  const [selectedId, setSelectedId] = useState(weapons[0]?.id || null);
  const [charging, setCharging] = useState(false);
  const [charge, setCharge] = useState(0);
  const chargeRef = useRef(null);

  const selected = weapons.find((item) => item.id === selectedId) || weapons[0];

  useEffect(() => {
    if (!open) {
      setCharging(false);
      setCharge(0);
    }
  }, [open]);

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

  const fire = () => {
    if (!selected || selected.count <= 0) return;
    const consume = charge >= 0.7 ? 3 : charge >= 0.35 ? 2 : 1;
    onConsume(selected.id, -consume);
    setCharge(0);
  };

  const handleFireClick = () => {
    if (charging) return;
    fire();
  };

  const handleChargeStart = () => {
    if (!selected || selected.count <= 0) return;
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
  }, [open, charging, selected]);

  return (
    <Modal open={open} onClose={onClose} className="attack-card">
      <header className="modal-header">
        <h3>Attack</h3>
      </header>
      <div className="attack-body">
        <ItemUseBar
          helpText="Fire weapon"
          actionLabel="FIRE"
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
          footerText={
            selected ? `Fire Weapon: ${selected.name} ${selected.level ? `L${selected.level}` : ''}` : '—'
          }
        />
      </div>
    </Modal>
  );
}
