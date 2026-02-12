import { runtime } from '../config/runtime.js';

const XMP_RADIUS = {
  1: 42,
  2: 48,
  3: 58,
  4: 72,
  5: 90,
  6: 112,
  7: 138,
  8: 168
};

const US_RADIUS = {
  1: 10,
  2: 13,
  3: 16,
  4: 18,
  5: 21,
  6: 24,
  7: 27,
  8: 30
};

const XMP_COST = {
  1: 50,
  2: 100,
  3: 150,
  4: 200,
  5: 250,
  6: 300,
  7: 350,
  8: 400
};

const US_COST = {
  1: 50,
  2: 100,
  3: 150,
  4: 200,
  5: 250,
  6: 300,
  7: 350,
  8: 400
};

const normalizeOverride = (override) => {
  if (!override || typeof override !== 'object') return null;
  const next = { XMP: { ...XMP_RADIUS }, US: { ...US_RADIUS } };
  ['XMP', 'US'].forEach((type) => {
    if (!override[type] || typeof override[type] !== 'object') return;
    Object.entries(override[type]).forEach(([level, value]) => {
      const lv = Number(level);
      const radius = Number(value);
      if (lv >= 1 && lv <= 8 && radius > 0) {
        next[type][lv] = radius;
      }
    });
  });
  return next;
};

const override = normalizeOverride(runtime.attackRadiusOverride);

const radiusTable = override || {
  XMP: XMP_RADIUS,
  US: US_RADIUS
};

const defaultAttackSpecs = {
  XMP: Object.fromEntries(
    Object.entries(XMP_RADIUS).map(([level, radiusM]) => [
      Number(level),
      { radiusM: Number(radiusM), costXm: Number(XMP_COST[level] || 0) }
    ])
  ),
  US: Object.fromEntries(
    Object.entries(US_RADIUS).map(([level, radiusM]) => [
      Number(level),
      { radiusM: Number(radiusM), costXm: Number(US_COST[level] || 0) }
    ])
  )
};

const readSpecByLevel = (source, level) => {
  if (!source || typeof source !== 'object') return null;
  return source[level] || source[String(level)] || null;
};

export const normalizeAttackSpecs = (raw) => {
  if (!raw || typeof raw !== 'object') return null;
  const next = { XMP: {}, US: {} };
  ['XMP', 'US'].forEach((type) => {
    const source = raw[type] || raw[String(type).toLowerCase()];
    for (let level = 1; level <= 8; level += 1) {
      const row = readSpecByLevel(source, level);
      if (!row || typeof row !== 'object') continue;
      const radiusM = Number(row.radiusM ?? row.radius ?? row.radius_m);
      const costXm = Number(row.costXm ?? row.cost_xm ?? row.cost);
      if (!Number.isFinite(radiusM) || radiusM <= 0 || !Number.isFinite(costXm) || costXm <= 0) continue;
      next[type][level] = { radiusM, costXm };
    }
  });
  const hasAny =
    Object.keys(next.XMP).length > 0 ||
    Object.keys(next.US).length > 0;
  return hasAny ? next : null;
};

export const resolveAttackSpec = (weaponType, weaponLevel, attackSpecs) => {
  const type = String(weaponType || '').toUpperCase();
  const level = Number(weaponLevel || 1);
  if (!['XMP', 'US'].includes(type) || level < 1 || level > 8) return null;
  const normalized = normalizeAttackSpecs(attackSpecs);
  const fromServer = readSpecByLevel(normalized?.[type], level);
  if (fromServer) return fromServer;
  return defaultAttackSpecs[type][level] || null;
};

export const weaponForItem = (item) => {
  const subtype = String(item?.subtype || '').toUpperCase();
  if (subtype === 'XMP' || subtype === 'US') {
    return { weaponType: subtype, weaponLevel: Number(item?.level || 1) };
  }
  return null;
};

export const attackRadiusForWeapon = (weaponType, weaponLevel, attackSpecs = null) => {
  const spec = resolveAttackSpec(weaponType, weaponLevel, attackSpecs);
  if (spec?.radiusM) return Number(spec.radiusM || 0);
  const type = String(weaponType || '').toUpperCase();
  const level = Number(weaponLevel || 1);
  const bucket = radiusTable[type];
  if (!bucket) return 0;
  return Number(bucket[level] || 0);
};

export const attackCostForWeapon = (weaponType, weaponLevel, attackSpecs = null) => {
  const spec = resolveAttackSpec(weaponType, weaponLevel, attackSpecs);
  if (spec?.costXm) return Number(spec.costXm || 0);
  const level = Number(weaponLevel || 1);
  if (!Number.isFinite(level) || level <= 0) return 0;
  return Math.max(0, Math.round(level * 50));
};

export const chargeDurationMs = (weaponLevel) => {
  const level = Math.max(1, Math.min(8, Number(weaponLevel || 1)));
  const maxMs = 1600;
  const minMs = 900;
  const ratio = (level - 1) / 7;
  return Math.round(maxMs + (minMs - maxMs) * ratio);
};

export const chargeProgress = (nowMs, startedAtMs, durationMs) => {
  const now = Number(nowMs);
  const startedAt = Number(startedAtMs);
  const duration = Math.max(1, Number(durationMs || 1));
  if (!Number.isFinite(now) || !Number.isFinite(startedAt)) return 0;
  return Math.max(0, Math.min(1, (now - startedAt) / duration));
};

export const chargeBonusForProgress = (progress) => {
  const p = Math.max(0, Math.min(1, Number(progress || 0)));
  return Number((p * 0.2).toFixed(4));
};

export const distanceMeters = (a, b) => {
  const lat1 = Number(a?.latitude);
  const lon1 = Number(a?.longitude);
  const lat2 = Number(b?.latitude);
  const lon2 = Number(b?.longitude);
  if (![lat1, lon1, lat2, lon2].every(Number.isFinite)) return Number.POSITIVE_INFINITY;

  const toRad = (deg) => (deg * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLon = toRad(lon2 - lon1);
  const x1 = toRad(lat1);
  const x2 = toRad(lat2);
  const sinLat = Math.sin(dLat / 2);
  const sinLon = Math.sin(dLon / 2);
  const c = sinLat * sinLat + Math.cos(x1) * Math.cos(x2) * sinLon * sinLon;
  return 6371000 * 2 * Math.asin(Math.min(1, Math.sqrt(c)));
};

export const attackCandidates = ({ player, portals, weaponType, weaponLevel, playerFaction, attackSpecs = null }) => {
  const radius = attackRadiusForWeapon(weaponType, weaponLevel, attackSpecs);
  if (!radius || !player) return [];
  const me = {
    latitude: player.latitude,
    longitude: player.longitude
  };

  return (portals || [])
    .map((portal) => {
      const d = distanceMeters(me, portal.position);
      return {
        portalId: portal.id,
        distanceMeters: d,
        portal
      };
    })
    .filter((it) => Number.isFinite(it.distanceMeters) && it.distanceMeters <= radius)
    .filter((it) => {
      const faction = String(it.portal?.faction || 'NEUTRAL').toUpperCase();
      if (faction === 'NEUTRAL') return false;
      if (!playerFaction) return true;
      return faction !== String(playerFaction).toUpperCase();
    })
    .sort((a, b) => a.distanceMeters - b.distanceMeters);
};

export const attackItemType = (weaponType, weaponLevel) => {
  const type = String(weaponType || '').toUpperCase();
  const level = Number(weaponLevel || 1);
  if (type === 'XMP') return `XMP_L${level}`;
  if (type === 'US') return `US_L${level}`;
  return '';
};
