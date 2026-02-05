const BASE = '/icons/item';

const rarityMap = {
  C: 'common',
  COMMON: 'common',
  R: 'rare',
  RARE: 'rare',
  VR: 'very-rare',
  VERY_RARE: 'very-rare',
  'VERY RARE': 'very-rare',
  'VERY-RARE': 'very-rare'
};

const normalizeRarity = (rarity) => {
  if (!rarity) return null;
  const key = String(rarity).toUpperCase();
  return rarityMap[key] || null;
};

const iconPath = (kind, name) => `${BASE}/${kind}/${name}`;

const pickItoVariant = (mod) => {
  const variant = String(mod?.variant || mod?.direction || mod?.orientation || 'up').toLowerCase();
  return variant === 'down' ? 'down' : 'up';
};

const isAegis = (rarity) => String(rarity || '').toUpperCase() === 'AXA';

const isSoftbank = (rarity) => String(rarity || '').toUpperCase() === 'SBUL';

const iconForMod = (mod, kind) => {
  if (!mod) return null;
  const subtype = String(mod.subtype || '').toUpperCase();
  const rarity = String(mod.rarity || '').toUpperCase();
  switch (subtype) {
    case 'SHIELD': {
      if (isAegis(rarity)) return iconPath(kind, 'portal-shield-aegis.png');
      const suffix = normalizeRarity(rarity);
      return suffix ? iconPath(kind, `portal-shield-${suffix}.png`) : null;
    }
    case 'LINK': {
      const suffix = normalizeRarity(rarity);
      if (!suffix || suffix === 'common') return null;
      return iconPath(kind, `link-amp-${suffix}.png`);
    }
    case 'SBUL':
      return iconPath(kind, 'softbank-ultra-link.png');
    case 'HEATSINK': {
      const suffix = normalizeRarity(rarity);
      return suffix ? iconPath(kind, `heat-sink-${suffix}.png`) : null;
    }
    case 'MULTI': {
      const suffix = normalizeRarity(rarity);
      return suffix ? iconPath(kind, `multi-hack-${suffix}.png`) : null;
    }
    case 'FORCE':
      return iconPath(kind, 'force-amp.png');
    case 'TURRET':
      return iconPath(kind, 'turret.png');
    case 'ITO':
    case 'ITO_EN_TRANSMUTER':
    case 'ITO_EN':
      return iconPath(kind, `ito-en-transmuter-${pickItoVariant(mod)}.png`);
    default:
      return null;
  }
};

export const getInventoryIcon = (item) => {
  if (!item) return null;
  const type = String(item.type || '').toLowerCase();
  const subtype = String(item.subtype || '').toUpperCase();
  const level = item.level;
  const name = String(item.name || '');
  const id = String(item.id || '');
  const label = `${name} ${id}`.toLowerCase();

  if (type === 'weapon') {
    if (subtype === 'XMP' && level) return iconPath('inventory', `xmp-burster-l${level}.png`);
    if (subtype === 'US' && level) return iconPath('inventory', `ultra-strike-l${level}.png`);
    if (subtype === 'VIRUS') {
      if (label.includes('ada')) return iconPath('inventory', 'ada-refactor.png');
      if (label.includes('jarvis')) return iconPath('inventory', 'jarvis-virus.png');
    }
    return null;
  }

  if (type === 'cube') {
    if (label.includes('hypercube') || subtype === 'HYPER' || subtype === 'HYPERCUBE') {
      return iconPath('inventory', 'hypercube.png');
    }
    if (subtype === 'CUBE' && level) return iconPath('inventory', `power-cube-l${level}.png`);
    return null;
  }

  if (type === 'resonator') {
    return level ? iconPath('inventory', `resonator-l${level}.png`) : null;
  }

  if (type === 'mod') {
    if (isSoftbank(item.rarity)) return iconPath('inventory', 'softbank-ultra-link.png');
    return iconForMod(item, 'inventory');
  }

  if (type === 'key') {
    return iconPath('inventory', 'portal-key.png');
  }

  return null;
};

export const getInstalledModIcon = (mod) => {
  if (!mod || !mod.subtype) return iconPath('installed', 'empty.png');
  if (isSoftbank(mod.rarity)) return iconPath('installed', 'softbank-ultra-link.png');
  const icon = iconForMod(mod, 'installed');
  return icon || iconPath('installed', 'empty.png');
};
