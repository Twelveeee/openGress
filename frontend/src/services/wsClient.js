const now = () => Date.now();

const randomId = () =>
  `${Math.random().toString(16).slice(2)}-${Math.random().toString(16).slice(2)}-${Date.now()}`;

export function createWsClient({
  url,
  getAuthToken,
  onOpen,
  onClose,
  onMessage,
  onError,
  reconnect = true
}) {
  let ws = null;
  let closedByUser = false;
  let retryCount = 0;
  let reconnectTimer = null;
  let pingTimer = null;
  const pendingById = new Map();

  const clearTimers = () => {
    if (reconnectTimer) clearTimeout(reconnectTimer);
    if (pingTimer) clearInterval(pingTimer);
    reconnectTimer = null;
    pingTimer = null;
  };

  const resolvePending = (id, payload, isError = false) => {
    const pending = pendingById.get(id);
    if (!pending) return;
    pendingById.delete(id);
    clearTimeout(pending.timer);
    if (isError) {
      pending.reject(payload);
    } else {
      pending.resolve(payload);
    }
  };

  const scheduleReconnect = () => {
    if (!reconnect || closedByUser) return;
    const delay = Math.min(1000 * Math.pow(2, retryCount), 10000);
    retryCount += 1;
    reconnectTimer = setTimeout(connect, delay);
  };

  const sendRaw = (payload) => {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      throw new Error('ws not connected');
    }
    ws.send(JSON.stringify(payload));
    return payload.id;
  };

  const send = (type, data = {}, id = randomId()) => {
    const payload = {
      type,
      timestamp: now(),
      id,
      data
    };
    sendRaw(payload);
    return id;
  };

  const request = (type, data = {}, timeout = 1000) => {
    const id = send(type, data);
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        pendingById.delete(id);
        resolve({ type: 'TIMEOUT', id, data: null });
      }, timeout);
      pendingById.set(id, { resolve, reject, timer });
    });
  };

  const handleOpen = () => {
    retryCount = 0;
    onOpen?.();
    const token = getAuthToken?.();
    if (token) {
      send('CONNECT', { authToken: token, version: 'frontend-v1' });
    }
    pingTimer = setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        send('PING', {});
      }
    }, 25000);
  };

  const handleClose = (event) => {
    clearTimers();
    onClose?.(event);
    scheduleReconnect();
  };

  const handleMessage = (event) => {
    let message = null;
    try {
      message = JSON.parse(event.data);
    } catch (_err) {
      return;
    }

    if (message?.id && pendingById.has(message.id)) {
      if (message.type === 'ERROR') {
        resolvePending(message.id, message, true);
        return;
      } else {
        resolvePending(message.id, message, false);
      }
    }

    onMessage?.(message);
  };

  const handleSocketError = (event) => {
    onError?.(event);
  };

  const connect = () => {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return;
    }
    closedByUser = false;
    ws = new WebSocket(url);
    ws.onopen = handleOpen;
    ws.onclose = handleClose;
    ws.onmessage = handleMessage;
    ws.onerror = handleSocketError;
  };

  const disconnect = () => {
    closedByUser = true;
    clearTimers();
    pendingById.forEach((pending) => {
      clearTimeout(pending.timer);
      pending.reject(new Error('ws disconnected'));
    });
    pendingById.clear();
    if (ws) {
      ws.close();
      ws = null;
    }
  };

  return {
    connect,
    disconnect,
    send,
    request,
    status: () => ws?.readyState ?? WebSocket.CLOSED
  };
}
