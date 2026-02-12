const levelColorClass = (level) => {
  if (level <= 3) return 'lvl-low';
  if (level <= 6) return 'lvl-mid';
  return 'lvl-high';
};

const formatModSubtype = (modType) => {
  const type = String(modType || '').toUpperCase();
  if (type.includes('LINK_AMP')) return 'LINK';
  if (type.includes('SOFTBANK')) return 'SBUL';
  if (type.includes('PORTAL_SHIELD') || type === 'SHIELD') return 'SHIELD';
  if (type.includes('FORCE_AMP')) return 'FORCE';
  if (type.includes('HEAT_SINK')) return 'HEATSINK';
  if (type.includes('MULTI_HACK')) return 'MULTI';
  if (type.includes('TURRET')) return 'TURRET';
  if (type.includes('ITO_EN')) return 'ITO_EN';
  return type;
};

const maxResonatorEnergyByLevel = {
  1: 1000,
  2: 1500,
  3: 2000,
  4: 2500,
  5: 3000,
  6: 4000,
  7: 5000,
  8: 6000
};

const toResonatorXM = (level, energy) => {
  const lvl = Number(level || 1);
  const eng = Number(energy || 0);
  const max = maxResonatorEnergyByLevel[lvl] || 1000;
  if (!Number.isFinite(eng) || max <= 0) return 0;
  return Math.max(0, Math.min(100, Math.round((eng / max) * 100)));
};

export const mapPlayer = (player) => {
  if (!player) return null;
  const latitude = Number(player.position?.latitude ?? player.latitude ?? 39.908722);
  const longitude = Number(player.position?.longitude ?? player.longitude ?? 116.397499);
  const renderLatitude = Number(player.renderLatitude ?? latitude);
  const renderLongitude = Number(player.renderLongitude ?? longitude);
  const preRenderLatitude = Number(player.preRenderLatitude ?? renderLatitude);
  const preRenderLongitude = Number(player.preRenderLongitude ?? renderLongitude);
  const targetLatitude = Number(player.target?.latitude ?? player.targetLatitude);
  const targetLongitude = Number(player.target?.longitude ?? player.targetLongitude);
  const hasTarget = Number.isFinite(targetLatitude) && Number.isFinite(targetLongitude);
  return {
    id: player.id,
    username: player.username,
    faction: player.faction || 'NEUTRAL',
    level: Number(player.level || 1),
    ap: Number(player.ap || 0),
    xm: Number(player.xm || 0),
    maxXm: Number(player.maxXm || 0),
    autoHack: Boolean(player.autoHack),
    speedMps: Number(player.speedMps || 0),
    headingDeg: Number(player.headingDeg || 0),
    latitude,
    longitude,
    renderLatitude,
    renderLongitude,
    preRenderLatitude,
    preRenderLongitude,
    target: hasTarget ? { latitude: targetLatitude, longitude: targetLongitude } : null,
    targetLatitude: hasTarget ? targetLatitude : null,
    targetLongitude: hasTarget ? targetLongitude : null
  };
};

export const mapPortalEntity = (portal) => {
  const resonatorMap = portal?.resonators || {};
  const modMap = portal?.mods || {};
  const resonators = Array.from({ length: 8 }, (_, i) => {
    const entry = resonatorMap[String(i + 1)] || resonatorMap[i + 1] || null;
    if (!entry) return null;
    return {
      slot: i + 1,
      level: Number(entry.level || 1),
      owner: entry.playerId || '',
      faction: portal?.faction || 'NEUTRAL',
      xm: toResonatorXM(entry.level, entry.energy),
      version: Number(entry.version || 0)
    };
  });

  const mods = Array.from({ length: 4 }, (_, i) => {
    const entry = modMap[String(i + 1)] || modMap[i + 1] || null;
    if (!entry) return null;
    return {
      slot: i + 1,
      type: String(entry.modType || '').replace(/_/g, ' '),
      subtype: formatModSubtype(entry.modType),
      rarity: entry.rarity || 'C',
      owner: entry.playerId || ''
    };
  });

  return {
    id: portal?.id,
    title: portal?.title || portal?.name || portal?.id || 'Portal',
    name: portal?.title || portal?.name || portal?.id || 'Portal',
    cover_url: portal?.cover_url || portal?.coverUrl || portal?.image || '',
    image: portal?.cover_url || portal?.coverUrl || portal?.image || '',
    faction: portal?.faction || 'NEUTRAL',
    level: Number(portal?.level || 1),
    energy: Number(portal?.energy || 0),
    position: {
      latitude: Number(portal?.position?.latitude ?? portal?.latitude ?? portal?.lat ?? 0),
      longitude: Number(portal?.position?.longitude ?? portal?.longitude ?? portal?.lng ?? 0)
    },
    lat: Number(portal?.position?.latitude ?? portal?.latitude ?? portal?.lat ?? 0),
    lng: Number(portal?.position?.longitude ?? portal?.longitude ?? portal?.lng ?? 0),
    owner: resonators.find((item) => item?.owner)?.owner || '',
    resonators,
    mods
  };
};

export const mapLinkEntity = (link) => ({
  id: link.id || link.linkId || `${link.fromPortalId}-${link.toPortalId}`,
  fromPortalId: link.fromPortalId,
  toPortalId: link.toPortalId,
  faction: link.faction || 'NEUTRAL',
  from: {
    lat: Number(link.fromPosition?.latitude ?? link.fromLat ?? 0),
    lng: Number(link.fromPosition?.longitude ?? link.fromLon ?? 0)
  },
  to: {
    lat: Number(link.toPosition?.latitude ?? link.toLat ?? 0),
    lng: Number(link.toPosition?.longitude ?? link.toLon ?? 0)
  }
});

export const mapFieldEntity = (field, portalMap = {}) => {
  const points = (field.portalIds || [])
    .map((portalId) => portalMap[portalId]?.position)
    .filter(Boolean)
    .map((pos) => ({ lat: Number(pos.latitude), lng: Number(pos.longitude) }));

  return {
    id: field.id || field.fieldId,
    portalIds: field.portalIds || [],
    faction: field.faction || 'NEUTRAL',
    mu: Number(field.mu || 0),
    layer: Number(field.layer || 1),
    points
  };
};

const keyMeta = (itemType) => {
  const portalId = String(itemType || '').split(':')[1] || '';
  return {
    id: itemType,
    name: portalId || 'Portal Key',
    type: 'key',
    subtype: 'KEY',
    level: 1,
    faction: 'NEUTRAL',
    portalId
  };
};

const parseItemType = (itemType) => {
  if (itemType.startsWith('RESO_L')) {
    const level = Number(itemType.replace('RESO_L', '')) || 1;
    return { id: itemType, name: 'Resonator', type: 'resonator', subtype: 'RES', level };
  }
  if (itemType.startsWith('XMP_L')) {
    const level = Number(itemType.replace('XMP_L', '')) || 1;
    return { id: itemType, name: 'XMP Burster', type: 'weapon', subtype: 'XMP', level };
  }
  if (itemType.startsWith('US_L')) {
    const level = Number(itemType.replace('US_L', '')) || 1;
    return { id: itemType, name: 'Ultra Strike', type: 'weapon', subtype: 'US', level };
  }
  if (itemType === 'CUBE' || itemType.startsWith('CUBE_L')) {
    const level = itemType === 'CUBE' ? 1 : Number(itemType.replace('CUBE_L', '')) || 1;
    return { id: itemType, name: 'Power Cube', type: 'cube', subtype: 'CUBE', level };
  }
  if (itemType.startsWith('MOD:')) {
    const [, modType = '', rarity = 'C'] = itemType.split(':');
    const subtype = formatModSubtype(modType);
    return {
      id: itemType,
      modType,
      name: modType.replace(/_/g, ' '),
      type: 'mod',
      subtype,
      rarity
    };
  }
  if (itemType.startsWith('KEY:')) {
    return keyMeta(itemType);
  }
  if (itemType === 'ADA') {
    return { id: itemType, name: 'ADA Refactor', type: 'weapon', subtype: 'VIRUS', rarity: 'VR' };
  }
  if (itemType === 'JARVIS') {
    return { id: itemType, name: 'Jarvis Virus', type: 'weapon', subtype: 'VIRUS', rarity: 'VR' };
  }
  return { id: itemType, name: itemType, type: 'weapon', subtype: 'XMP', level: 1 };
};

export const mapInventory = (inventoryMap, portalLookup = {}) => {
  const entries = Object.entries(inventoryMap || {});
  return entries
    .map(([itemType, amount]) => {
      const base = parseItemType(itemType);
      if (base.type === 'key' && base.portalId) {
        const portal = portalLookup[base.portalId];
        if (portal) {
          return {
            ...base,
            name: portal.name || base.name,
            faction: portal.faction || 'NEUTRAL',
            level: portal.level || 1,
            resonators: (portal.resonators || []).map((slot) => slot?.xm || 0)
          };
        }
      }
      return base;
    })
    .map((item) => ({ ...item, count: Number(amountFromMap(inventoryMap, item.id)) }))
    .filter((item) => item.count > 0)
    .sort((a, b) => {
      if (a.type !== b.type) return a.type.localeCompare(b.type);
      if ((a.level || 0) !== (b.level || 0)) return (a.level || 0) - (b.level || 0);
      return a.id.localeCompare(b.id);
    });
};

const amountFromMap = (inventoryMap, id) => Number(inventoryMap?.[id] || 0);

export const apProgress = (ap, level) => {
  const lvl = Number(level || 1);
  const table = {
    1: 0,
    2: 2500,
    3: 20000,
    4: 70000,
    5: 150000,
    6: 300000,
    7: 600000,
    8: 1200000,
    9: 2400000,
    10: 4000000,
    11: 6000000,
    12: 8400000,
    13: 12000000,
    14: 17000000,
    15: 24000000,
    16: 40000000
  };
  const currentFloor = table[lvl] ?? 0;
  const next = table[lvl + 1];
  if (!next) {
    return {
      value: Number(ap || 0),
      min: currentFloor,
      max: currentFloor,
      percent: 100,
      text: `${Number(ap || 0)} AP`
    };
  }
  const value = Number(ap || 0);
  const percent = Math.max(0, Math.min(100, ((value - currentFloor) / (next - currentFloor)) * 100));
  return {
    value,
    min: currentFloor,
    max: next,
    percent,
    text: `${value} / ${next} AP`
  };
};

export const formatWsLog = (type, message) => {
  const ts = new Date(message?.timestamp || Date.now());
  const hh = String(ts.getHours()).padStart(2, '0');
  const mm = String(ts.getMinutes()).padStart(2, '0');
  const data = message?.data || {};
  let text = data.message || data.portalId || '';
  if (type === 'ATTACK') {
    const weaponType = String(data.weaponType || '').toUpperCase();
    const weaponLevel = Number(data.weaponLevel || 0);
    const portalId = String(data.portalId || '');
    const damage = Number(data.damage ?? data.damageDealt ?? 0);
    const resonatorsDestroyed = Number(data.resonatorsDestroyed || 0);
    const modsDestroyed = Array.isArray(data.modsDestroyed) ? data.modsDestroyed.length : 0;
    const portalDamages = Array.isArray(data.portalDamages) ? data.portalDamages.length : 0;
    const parts = [];
    if (weaponType && weaponLevel > 0) {
      parts.push(`${weaponType} L${weaponLevel}`);
    }
    if (portalId) {
      parts.push(`@${portalId}`);
    }
    parts.push(`DMG ${damage}`);
    parts.push(`RESO ${resonatorsDestroyed}`);
    if (modsDestroyed > 0) {
      parts.push(`MOD -${modsDestroyed}`);
    }
    if (portalDamages > 0) {
      parts.push(`P${portalDamages}`);
    }
    if (data.counterattackTriggered || data.counterattack) {
      parts.push(`COUNTER -${Number(data.counterattackDamage || 0)} XM`);
    }
    text = parts.join(' · ');
  }
  return {
    id: `${type}-${message?.id || Math.random().toString(16).slice(2)}`,
    time: `${hh}:${mm}`,
    timestamp: ts.getTime(),
    type,
    text,
    message: text,
    playerId: data.playerId || '',
    portalId: data.portalId || '',
    faction: data.faction || '',
    mu: Number(data.mu || 0)
  };
};

export const portalPatchFromUpdate = (update = {}) => {
  const patch = {};
  if (update.portalId) patch.id = update.portalId;
  if (update.faction) patch.faction = update.faction;
  if (update.portalLevel != null) patch.level = Number(update.portalLevel || 1);
  if (update.level != null) patch.level = Number(update.level || 1);
  if (update.energy != null) patch.energy = Number(update.energy || 0);
  if (update.owner) patch.owner = String(update.owner || '');

  if (update.resonators != null) {
    patch.resonatorsSnapshot = buildResonatorSnapshot(update.resonators, patch.faction || update.faction);
  }
  if (update.mods != null) {
    patch.modsSnapshot = buildModSnapshot(update.mods);
  }

  if (update.slot) {
    const updateVersion = Number(update.version);
    const slotEnergy = Number(update.slotEnergy);
    patch.resonatorUpdate = {
      slot: Number(update.slot),
      level: Number(update.level || 1),
      energy: Number.isFinite(slotEnergy) ? slotEnergy : null,
      owner: update.playerId || '',
      version: Number.isFinite(updateVersion) ? updateVersion : null
    };
  }
  if (update.modSlot) {
    patch.modUpdate = {
      slot: Number(update.modSlot),
      modType: update.modType,
      rarity: update.rarity,
      owner: update.playerId || ''
    };
  }
  return patch;
};

export const applyPortalPatch = (portal, patch) => {
  if (!portal) return portal;
  const next = { ...portal };
  if (patch.faction) next.faction = patch.faction;
  if (patch.level != null) next.level = patch.level;
  if (patch.energy != null) next.energy = patch.energy;
  if (patch.owner) next.owner = patch.owner;

  if (patch.resonatorsSnapshot) {
    next.resonators = patch.resonatorsSnapshot;
    if (!patch.owner) {
      next.owner = patch.resonatorsSnapshot.find((item) => item?.owner)?.owner || '';
    }
  }
  if (patch.modsSnapshot) {
    next.mods = patch.modsSnapshot;
  }

  if (patch.resonatorUpdate) {
    const slots = [...(next.resonators || Array.from({ length: 8 }, () => null))];
    const idx = patch.resonatorUpdate.slot - 1;
    if (idx >= 0 && idx < 8) {
      const prevSlot = slots[idx] || {};
      const level = Number(patch.resonatorUpdate.level || prevSlot.level || 1);
      const owner = patch.resonatorUpdate.owner || prevSlot.owner || '';
      let xm = Number(prevSlot.xm || 0);
      if (Number.isFinite(patch.resonatorUpdate.energy)) {
        xm = toResonatorXM(level, patch.resonatorUpdate.energy);
      } else if (level > 0) {
        xm = Math.max(100, xm);
      }
      slots[idx] = {
        slot: patch.resonatorUpdate.slot,
        level,
        owner,
        faction: next.faction,
        xm,
        version:
          Number.isFinite(patch.resonatorUpdate.version) && patch.resonatorUpdate.version >= 0
            ? Number(patch.resonatorUpdate.version)
            : Number(prevSlot.version || 0)
      };
    }
    next.resonators = slots;
    next.owner = slots.find((item) => item?.owner)?.owner || '';
  }

  if (patch.modUpdate) {
    const slots = [...(next.mods || Array.from({ length: 4 }, () => null))];
    const idx = patch.modUpdate.slot - 1;
    if (idx >= 0 && idx < 4) {
      slots[idx] = {
        slot: patch.modUpdate.slot,
        type: String(patch.modUpdate.modType || '').replace(/_/g, ' '),
        subtype: formatModSubtype(patch.modUpdate.modType),
        rarity: patch.modUpdate.rarity || 'C',
        owner: patch.modUpdate.owner
      };
    }
    next.mods = slots;
  }
  return next;
};

const readSnapshotSlot = (source, slot) => {
  if (Array.isArray(source)) {
    return (
      source.find((item) => Number(item?.slot) === slot) ||
      source[slot - 1] ||
      source[slot] ||
      null
    );
  }
  if (!source || typeof source !== 'object') return null;
  return source[String(slot)] || source[slot] || null;
};

const buildResonatorSnapshot = (source, faction) =>
  Array.from({ length: 8 }, (_, index) => {
    const slot = index + 1;
    const entry = readSnapshotSlot(source, slot);
    if (!entry) return null;
    const level = Number(entry.level || 0);
    if (level <= 0) return null;
    const energy = Number(entry.energy || 0);
    return {
      slot,
      level,
      owner: String(entry.playerId || entry.owner || ''),
      faction: faction || 'NEUTRAL',
      xm: toResonatorXM(level, energy),
      version: Number(entry.version || 0)
    };
  });

const buildModSnapshot = (source) =>
  Array.from({ length: 4 }, (_, index) => {
    const slot = index + 1;
    const entry = readSnapshotSlot(source, slot);
    if (!entry) return null;
    const modType = String(entry.modType || entry.type || '').toUpperCase();
    if (!modType) return null;
    return {
      slot,
      type: modType.replace(/_/g, ' '),
      subtype: formatModSubtype(modType),
      rarity: entry.rarity || 'C',
      owner: entry.playerId || entry.owner || ''
    };
  });

export { levelColorClass };
