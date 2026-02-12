import React, { useEffect, useMemo, useRef, useState } from 'react';
import Modal from '../components/Modal.jsx';
import ItemUseBar from '../components/ItemUseBar.jsx';
import {
  attackCostForWeapon,
  chargeBonusForProgress,
  chargeDurationMs
} from '../constants/attackProfiles.js';

function sortWeapons(a, b) {
  const order = { XMP: 1, US: 2 };
  const aType = order[String(a.subtype || '').toUpperCase()] || 99;
  const bType = order[String(b.subtype || '').toUpperCase()] || 99;
  if (aType !== bType) return aType - bType;
  return Number(a.level || 0) - Number(b.level || 0);
}

export default function AttackModal({
  open,
  onClose,
  items = [],
  playerXm = 0,
  attackSpecs = null,
  onAttack,
  onChargeFxChange,
  onAttackError
}) {
  const weapons = useMemo(
    () =>
      items
        .filter((item) => item.type === 'weapon' && ['XMP', 'US'].includes(String(item.subtype || '').toUpperCase()))
        .sort(sortWeapons),
    [items]
  );
  const displayWeapons = useMemo(() => {
    if (weapons.length) return weapons;
    return [
      {
        id: 'WEAPON_EMPTY',
        name: 'No Weapon',
        type: 'placeholder',
        subtype: 'NONE',
        level: 0,
        count: 0,
        placeholder: true
      }
    ];
  }, [weapons]);

  const [selectedId, setSelectedId] = useState(weapons[0]?.id || null);
  const [charging, setCharging] = useState(false);
  const [charge, setCharge] = useState(0);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const chargeRef = useRef(null);
  const chargeValueRef = useRef(0);
  const chargeDurationRef = useRef(1200);
  const suppressClickRef = useRef(false);

  const selected = weapons.find((item) => item.id === selectedId) || weapons[0];
  const selectedWeapon = selected && !selected?.placeholder ? selected : null;
  const selectedWeaponType = String(selectedWeapon?.subtype || '').toUpperCase();
  const selectedWeaponLevel = Number(selectedWeapon?.level || 1);
  const requiredXm = selectedWeapon
    ? attackCostForWeapon(selectedWeaponType, selectedWeaponLevel, attackSpecs)
    : 0;
  const currentXm = Number(playerXm || 0);
  const xmEnough = requiredXm <= 0 || currentXm >= requiredXm;
  const xmNotice =
    selectedWeapon && !xmEnough ? `XM不足（需要 ${requiredXm}，当前 ${Math.max(0, Math.floor(currentXm))}）` : '';

  useEffect(() => {
    if (!open) {
      setCharging(false);
      setCharge(0);
      chargeValueRef.current = 0;
      onChargeFxChange?.(null);
      setBusy(false);
      setNotice('');
      return;
    }
    if (!selected && weapons[0]) {
      setSelectedId(weapons[0].id);
    }
  }, [onChargeFxChange, open, selected, weapons]);

  useEffect(() => {
    if (!charging) return undefined;
    const start = performance.now();
    const duration = Math.max(1, Number(chargeDurationRef.current || 1));

    const tick = (now) => {
      const t = Math.min((now - start) / duration, 1);
      setCharge(t);
      chargeValueRef.current = t;
      chargeRef.current = requestAnimationFrame(tick);
    };

    chargeRef.current = requestAnimationFrame(tick);
    return () => {
      if (chargeRef.current) cancelAnimationFrame(chargeRef.current);
    };
  }, [charging]);

  const fire = async (chargeRatio = 0) => {
    if (!selectedWeapon || busy) return;
    if (!xmEnough) {
      setNotice(xmNotice);
      onAttackError?.(xmNotice);
      return;
    }
    setBusy(true);
    const bonus = chargeBonusForProgress(chargeRatio);
    try {
      const result = await onAttack?.({ item: selected, charge: bonus });
      if (result?.ok) {
        setNotice(`已发射 · 命中 ${result.count || 0}`);
      } else {
        const errorText = String(result?.error || '发射失败');
        setNotice(errorText);
        onAttackError?.(errorText);
      }
    } finally {
      setBusy(false);
      setCharge(0);
      chargeValueRef.current = 0;
    }
  };

  const handleFireClick = () => {
    if (suppressClickRef.current) return;
    if (charging || busy) return;
    fire(0);
  };

  const handleChargeStart = () => {
    if (!selectedWeapon || busy || charging || !xmEnough) return;
    const durationMs = chargeDurationMs(selectedWeaponLevel);
    chargeDurationRef.current = durationMs;
    setCharging(true);
    chargeValueRef.current = 0;
    onChargeFxChange?.({
      active: true,
      startAt: Date.now(),
      durationMs,
      weaponLevel: selectedWeaponLevel
    });
  };

  const handleChargeEnd = () => {
    if (!charging) return;
    suppressClickRef.current = true;
    const ratio = chargeValueRef.current;
    setCharging(false);
    onChargeFxChange?.(null);
    fire(ratio);
    window.setTimeout(() => {
      suppressClickRef.current = false;
    }, 0);
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
  }, [busy, charging, open, selected, selectedWeapon, xmEnough]);

  return (
    <Modal
      open={open}
      onClose={onClose}
      className="attack-card"
      placement="bottom"
      nonBlocking
      closeOnBackdrop={false}
    >
      <header className="modal-header">
        <h3>Attack</h3>
      </header>
      <div className="attack-body">
        <ItemUseBar
          helpText="按住空格或长按 FIRE 蓄力，松开发射。加成从0%线性提升到20%，环越接近中心加成越高。"
          actionLabel={busy ? 'FIRING...' : 'FIRE'}
          items={displayWeapons}
          selectedId={selected?.id}
          onSelect={setSelectedId}
          onAction={handleFireClick}
          actionClassName={charging ? 'charging' : ''}
          onActionMouseDown={handleChargeStart}
          onActionMouseUp={handleChargeEnd}
          onActionMouseLeave={handleChargeEnd}
          onActionTouchStart={handleChargeStart}
          onActionTouchEnd={handleChargeEnd}
          onActionTouchCancel={handleChargeEnd}
          actionDisabled={!selectedWeapon || busy || !xmEnough}
          actionDisabledText={!selectedWeapon ? 'NO WEAPON' : !xmEnough ? 'XM LOW' : undefined}
          notice={notice}
          noticeKey={`${selected?.id || 'none'}-${notice}`}
          footerText={
            selectedWeapon
              ? `${selectedWeapon.name} L${selectedWeaponLevel}${charging ? ` · charge ${Math.round(charge * 100)}%` : ''}${xmNotice ? ` · ${xmNotice}` : ''}`
              : '无可用攻击道具'
          }
        />
      </div>
    </Modal>
  );
}
