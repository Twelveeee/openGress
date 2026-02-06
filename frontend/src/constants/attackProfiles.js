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

export const weaponForItem = (item) => {
  const subtype = String(item?.subtype || '').toUpperCase();
  if (subtype === 'XMP' || subtype === 'US') {
    return { weaponType: subtype, weaponLevel: Number(item?.level || 1) };
  }
  return null;
};

export const attackRadiusForWeapon = (weaponType, weaponLevel) => {
  const type = String(weaponType || '').toUpperCase();
  const level = Number(weaponLevel || 1);
  const bucket = radiusTable[type];
  if (!bucket) return 0;
  return Number(bucket[level] || 0);
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

export const attackCandidates = ({ player, portals, weaponType, weaponLevel, playerFaction }) => {
  const radius = attackRadiusForWeapon(weaponType, weaponLevel);
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
