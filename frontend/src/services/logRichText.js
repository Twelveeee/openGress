const normalizeFaction = (faction) => String(faction || '').trim().toUpperCase();

export const factionClass = (faction) => {
  const key = normalizeFaction(faction);
  if (key === 'RES' || key === 'RESISTANCE') return 'res';
  if (key === 'ENL' || key === 'ENLIGHTENED') return 'enl';
  return '';
};

const token = (text, kind = 'plain', faction = '', meta = {}) => {
  const value = String(text || '');
  if (!value) return null;
  const clsFaction = factionClass(faction);
  const className =
    kind === 'plain' ? '' : kind === 'portal' ? 'log-token-portal' : `log-token-${kind}${clsFaction ? ` ${clsFaction}` : ''}`;
  return {
    text: value,
    kind,
    className,
    entityType: meta.entityType || null,
    entityId: meta.entityId || '',
    clickable: Boolean(meta.clickable && meta.entityType && meta.entityId)
  };
};

const compact = (parts) => parts.filter(Boolean);

const resolvePlayerIDByName = (name, ctx = {}) => {
  const target = String(name || '').trim();
  if (!target) return '';
  const entries = Object.entries(ctx?.playerNameById || {}).filter(([, value]) => String(value || '').trim() === target);
  if (entries.length !== 1) return '';
  return String(entries[0][0] || '');
};

const resolvePortalIDByName = (name, ctx = {}) => {
  const target = String(name || '').trim();
  if (!target) return '';
  const ids = ctx?.portalIdByName?.[target];
  if (!Array.isArray(ids) || ids.length !== 1) return '';
  return String(ids[0] || '');
};

const playerToken = (name, faction, ctx = {}, preferredID = '') => {
  const id = String(preferredID || resolvePlayerIDByName(name, ctx) || '');
  return token(name, 'player', faction, { entityType: 'player', entityId: id, clickable: Boolean(id) });
};

const portalToken = (name, ctx = {}, preferredID = '') => {
  const id = String(preferredID || resolvePortalIDByName(name, ctx) || '');
  return token(name, 'portal', '', { entityType: 'portal', entityId: id, clickable: Boolean(id) });
};

const splitPlayerTag = (parts, faction, log = {}, ctx = {}) => {
  const out = [];
  const splitRe = /(<[^>]+>)/g;
  const playerRe = /^<[^>]+>$/;
  parts.forEach((part) => {
    if (!part || part.kind !== 'plain') {
      out.push(part);
      return;
    }
    const chunks = String(part.text || '').split(splitRe);
    chunks.forEach((chunk) => {
      if (!chunk) return;
      if (playerRe.test(chunk)) {
        const rawName = chunk.replace(/^</, '').replace(/>$/, '').trim();
        out.push(playerToken(chunk, faction, ctx, log.playerId || resolvePlayerIDByName(rawName, ctx)));
      } else {
        out.push(token(chunk, 'plain'));
      }
    });
  });
  return compact(out);
};

const splitByPlayer = (parts, faction, _log = {}, ctx = {}) => {
  const out = [];
  const re = /(\bby\s+)([^\s]+)\b/i;
  parts.forEach((part) => {
    if (!part || part.kind !== 'plain') {
      out.push(part);
      return;
    }
    const text = String(part.text || '');
    const match = text.match(re);
    if (!match || match.index == null) {
      out.push(part);
      return;
    }
    const start = match.index;
    const full = match[0];
    const prefix = text.slice(0, start);
    const byPrefix = match[1];
    const playerName = match[2];
    const suffix = text.slice(start + full.length);
    out.push(token(prefix, 'plain'));
    out.push(token(byPrefix, 'plain'));
    out.push(playerToken(playerName, faction, ctx));
    out.push(token(suffix, 'plain'));
  });
  return compact(out);
};

const parseLink = (text, faction, ctx = {}) => {
  let m = text.match(/^(.*?)(linked from )(.+?)( to )(.+)$/i);
  if (m) {
    const fromName = String(m[3] || '').trim();
    const toName = String(m[5] || '').trim();
    return compact([
      token(m[1], 'plain'),
      token(m[2], 'link', faction),
      portalToken(fromName, ctx),
      token(m[4], 'plain'),
      portalToken(toName, ctx)
    ]);
  }
  m = text.match(/^(.*?)(destroyed the Link )(.+?)( to )(.+)$/i);
  if (m) {
    const fromName = String(m[3] || '').trim();
    const toName = String(m[5] || '').trim();
    return compact([
      token(m[1], 'plain'),
      token(m[2], 'link', faction),
      portalToken(fromName, ctx),
      token(m[4], 'plain'),
      portalToken(toName, ctx)
    ]);
  }
  return null;
};

const parseField = (text, faction, ctx = {}, log = {}) => {
  let m = text.match(/^(.*?)(created a Control Field @.+)$/i);
  if (m) {
    const fieldChunk = String(m[2] || '');
    const portalMatch = fieldChunk.match(/@(.+?)(\s+[+-]?\d+\s+MUs?)?$/i);
    const portalName = String(portalMatch?.[1] || '').trim();
    const muSuffix = String(portalMatch?.[2] || '');
    const fieldPrefix = fieldChunk.slice(0, fieldChunk.indexOf('@') + 1);
    return compact([
      token(m[1], 'plain'),
      token(fieldPrefix, 'field', faction),
      portalToken(portalName, ctx, log.portalId),
      token(muSuffix, 'field', faction)
    ]);
  }
  m = text.match(/^(.*?)(destroyed the Control Field @.+)$/i);
  if (m) {
    const fieldChunk = String(m[2] || '');
    const portalMatch = fieldChunk.match(/@(.+?)(\s+[+-]?\d+\s+MUs?)?$/i);
    const portalName = String(portalMatch?.[1] || '').trim();
    const muSuffix = String(portalMatch?.[2] || '');
    const fieldPrefix = fieldChunk.slice(0, fieldChunk.indexOf('@') + 1);
    return compact([
      token(m[1], 'plain'),
      token(fieldPrefix, 'field', faction),
      portalToken(portalName, ctx, log.portalId),
      token(muSuffix, 'field', faction)
    ]);
  }
  return null;
};

const parsePortal = (text, ctx = {}, log = {}) => {
  let m = text.match(/^(.*?captured\s+)(.+)$/i);
  if (m) {
    return compact([token(m[1], 'plain'), portalToken(String(m[2] || '').trim(), ctx, log.portalId)]);
  }
  m = text.match(/^(.*?deployed a Resonator on\s+)(.+)$/i);
  if (m) {
    return compact([token(m[1], 'plain'), portalToken(String(m[2] || '').trim(), ctx, log.portalId)]);
  }
  m = text.match(/^(.*?Your Portal\s+)(.+?)(\s+is under attack.*)$/i);
  if (m) {
    return compact([
      token(m[1], 'plain'),
      portalToken(String(m[2] || '').trim(), ctx, log.portalId),
      token(m[3], 'plain')
    ]);
  }
  return null;
};

const resolvePlayerName = (playerID, ctx = {}) => {
  const id = String(playerID || '');
  if (!id) return '';
  const mapped = ctx?.playerNameById?.[id];
  return String(mapped || `<${id}>`);
};

const resolvePortalName = (portalID, ctx = {}) => {
  const id = String(portalID || '');
  if (!id) return '';
  const mapped = ctx?.portalNameById?.[id];
  return String(mapped || id);
};

const structuredTokens = (log = {}, ctx = {}) => {
  const type = String(log.type || '').toUpperCase();
  const message = String(log.message || log.text || '').trim().toLowerCase();
  const faction = log.faction || '';
  const playerID = String(log.playerId || '');
  const playerName = resolvePlayerName(playerID, ctx);
  const portalID = String(log.portalId || '');
  const portalName = resolvePortalName(portalID, ctx);
  const mu = Number(log.mu || 0);

  const withPlayerPrefix = (parts) => {
    if (!playerName) return parts;
    return compact([playerToken(playerName, faction, ctx, playerID), token(' ', 'plain'), ...parts]);
  };

  if (type === 'CAPTURE' || message === 'portal captured') {
    if (!portalName) return null;
    return withPlayerPrefix([token('captured ', 'plain'), portalToken(portalName, ctx, portalID)]);
  }
  if (type === 'FIELD' || message === 'field created') {
    if (!portalName) return null;
    const muSuffix = Number.isFinite(mu) && mu > 0 ? ` +${mu} MUs` : '';
    return withPlayerPrefix([
      token('created a Control Field @', 'field', faction),
      portalToken(portalName, ctx, portalID),
      token(muSuffix, 'field', faction)
    ]);
  }
  if (type === 'LINK' || message === 'link created') {
    if (!portalName) return null;
    return withPlayerPrefix([
      token('linked from ', 'link', faction),
      portalToken(portalName, ctx, portalID)
    ]);
  }
  return null;
};

export const buildLogTokens = (log = {}, ctx = {}) => {
  const text = String(log.message || log.text || '');
  const faction = log.faction || '';
  const structured = structuredTokens(log, ctx);
  if (structured) return structured;
  if (!text) return [];

  let parts =
    parseLink(text, faction, ctx) ||
    parseField(text, faction, ctx, log) ||
    parsePortal(text, ctx, log) ||
    [token(text, 'plain')];
  parts = splitPlayerTag(parts, faction, log, ctx);
  parts = splitByPlayer(parts, faction, log, ctx);
  return compact(parts);
};
