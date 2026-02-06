const trimSlash = (value) => String(value || '').replace(/\/+$/, '');

const inferApiBase = () => {
  if (typeof window === 'undefined') return '';
  return window.location.origin;
};

const inferWsBase = () => {
  if (typeof window === 'undefined') return '';
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
};

const parseAttackProtocol = (value) => {
  const raw = String(value || '').toLowerCase();
  if (raw === 'batch' || raw === 'single') return raw;
  return 'auto';
};

const parseAttackRadiusOverride = (value) => {
  if (!value) return null;
  try {
    const parsed = JSON.parse(value);
    return typeof parsed === 'object' && parsed ? parsed : null;
  } catch (_err) {
    return null;
  }
};

export const runtime = {
  apiBaseUrl: trimSlash(import.meta.env.VITE_API_BASE_URL) || inferApiBase(),
  wsUrl: trimSlash(import.meta.env.VITE_WS_URL) || `${inferWsBase()}/ws`,
  attackProtocol: parseAttackProtocol(import.meta.env.VITE_ATTACK_PROTOCOL),
  attackRadiusOverride: parseAttackRadiusOverride(import.meta.env.VITE_ATTACK_RADIUS_JSON)
};

export const apiPath = (path) => {
  const base = runtime.apiBaseUrl;
  const normalized = path.startsWith('/') ? path : `/${path}`;
  return `${base}${normalized}`;
};
