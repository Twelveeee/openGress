import { apiPath } from '../config/runtime.js';

const jsonHeaders = {
  'Content-Type': 'application/json'
};

const normalizeErr = (message, status = 0, errno = 1002) => ({
  errno,
  errmsg: message || 'request failed',
  status,
  data: null
});

export function createHttpClient({ getAccessToken, getRefreshToken, onAuthUpdate, onAuthInvalid }) {
  let refreshingPromise = null;

  const rawRequest = async (path, { method = 'GET', body, auth = true, signal } = {}) => {
    const headers = { ...jsonHeaders };
    if (auth) {
      const token = getAccessToken?.();
      if (token) {
        headers.Authorization = `Bearer ${token}`;
      }
    }

    const response = await fetch(apiPath(path), {
      method,
      headers,
      body: body == null ? undefined : JSON.stringify(body),
      signal
    });

    let payload = null;
    try {
      payload = await response.json();
    } catch (_err) {
      payload = null;
    }

    if (!payload || typeof payload !== 'object') {
      throw normalizeErr(`invalid response (${response.status})`, response.status, 1002);
    }

    const errno = Number(payload.errno ?? 1002);
    if (!response.ok || errno !== 0) {
      throw {
        errno,
        errmsg: payload.errmsg || `http ${response.status}`,
        status: response.status,
        data: payload.data ?? null
      };
    }

    return payload.data;
  };

  const refreshOnce = async () => {
    if (refreshingPromise) return refreshingPromise;
    const refreshToken = getRefreshToken?.();
    if (!refreshToken) {
      throw normalizeErr('missing refresh token', 401, 1004);
    }

    refreshingPromise = (async () => {
      const data = await rawRequest('/api/v1/auth/refresh', {
        method: 'POST',
        auth: false,
        body: { refresh_token: refreshToken }
      });
      const nextAccessToken = data?.access_token || '';
      if (!nextAccessToken) {
        throw normalizeErr('invalid refresh response', 401, 1004);
      }
      onAuthUpdate?.({ accessToken: nextAccessToken });
      return nextAccessToken;
    })();

    try {
      return await refreshingPromise;
    } finally {
      refreshingPromise = null;
    }
  };

  const request = async (path, options = {}, retry = true) => {
    try {
      return await rawRequest(path, options);
    } catch (err) {
      const shouldRetry =
        retry &&
        options.auth !== false &&
        Number(err?.errno) === 1004 &&
        !String(path).includes('/api/v1/auth/refresh');

      if (!shouldRetry) {
        if (Number(err?.errno) === 1004 && options.auth !== false) {
          onAuthInvalid?.();
        }
        throw err;
      }

      try {
        await refreshOnce();
      } catch (_refreshErr) {
        onAuthInvalid?.();
        throw err;
      }

      return request(path, options, false);
    }
  };

  return {
    get: (path, opts = {}) => request(path, { ...opts, method: 'GET' }),
    post: (path, body, opts = {}) => request(path, { ...opts, method: 'POST', body }),
    patch: (path, body, opts = {}) => request(path, { ...opts, method: 'PATCH', body })
  };
}
