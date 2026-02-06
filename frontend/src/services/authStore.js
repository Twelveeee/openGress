const KEY = 'opengress.auth.v1';

const safeRead = () => {
  if (typeof window === 'undefined') return null;
  try {
    const raw = window.localStorage.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') return null;
    return {
      accessToken: parsed.accessToken || '',
      refreshToken: parsed.refreshToken || '',
      playerId: parsed.playerId || ''
    };
  } catch (_err) {
    return null;
  }
};

const safeWrite = (value) => {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(KEY, JSON.stringify(value));
};

const safeRemove = () => {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem(KEY);
};

export const authStore = {
  get() {
    return safeRead();
  },
  set(tokens) {
    const next = {
      accessToken: tokens?.accessToken || '',
      refreshToken: tokens?.refreshToken || '',
      playerId: tokens?.playerId || ''
    };
    safeWrite(next);
    return next;
  },
  clear() {
    safeRemove();
  }
};
