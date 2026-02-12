import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import TopBar from './components/TopBar.jsx';
import MapShell from './components/MapShell.jsx';
import LogDock from './components/LogDock.jsx';
import NotifyLayer from './components/NotifyLayer.jsx';
import PortalModal from './modals/PortalModal.jsx';
import PortalDeployModal from './modals/PortalDeployModal.jsx';
import PortalModDeployModal from './modals/PortalModDeployModal.jsx';
import PlayerModal from './modals/PlayerModal.jsx';
import InventoryModal from './modals/InventoryModal.jsx';
import AttackModal from './modals/AttackModal.jsx';
import LeaderboardModal from './modals/LeaderboardModal.jsx';
import SettingsModal from './modals/SettingsModal.jsx';
import LayersModal from './modals/LayersModal.jsx';
import { runtime } from './config/runtime.js';
import { authStore } from './services/authStore.js';
import { createHttpClient } from './services/httpClient.js';
import { createWsClient } from './services/wsClient.js';
import {
  applyPortalPatch,
  apProgress,
  formatWsLog,
  mapFieldEntity,
  mapInventory,
  mapLinkEntity,
  mapPlayer,
  mapPortalEntity,
  portalPatchFromUpdate
} from './services/mappers.js';
import {
  attackRadiusForWeapon,
  normalizeAttackSpecs,
  weaponForItem
} from './constants/attackProfiles.js';

const initialModals = {
  portal: false,
  portalDeploy: false,
  portalModDeploy: false,
  player: false,
  inventory: false,
  attack: false,
  leaderboard: false,
  settings: false,
  layers: false
};

const defaultBounds = (player) => {
  const lat = Number(player?.latitude || 39.908722);
  const lng = Number(player?.longitude || 116.397499);
  return {
    minLat: lat - 0.01,
    maxLat: lat + 0.01,
    minLon: lng - 0.01,
    maxLon: lng + 0.01
  };
};

const parsePlayerTarget = (payload) => {
  const latitude = Number(
    payload?.target?.latitude ??
      payload?.targetLatitude ??
      payload?.targetLat ??
      payload?.destination?.latitude
  );
  const longitude = Number(
    payload?.target?.longitude ??
      payload?.targetLongitude ??
      payload?.targetLon ??
      payload?.destination?.longitude
  );
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
    return null;
  }
  return { latitude, longitude };
};

const haversineMeters = (a, b) => {
  const toRad = (deg) => (deg * Math.PI) / 180;
  const lat1 = toRad(a.latitude);
  const lat2 = toRad(b.latitude);
  const dLat = lat2 - lat1;
  const dLng = toRad(b.longitude - a.longitude);
  const sinLat = Math.sin(dLat / 2);
  const sinLng = Math.sin(dLng / 2);
  const h = sinLat * sinLat + Math.cos(lat1) * Math.cos(lat2) * sinLng * sinLng;
  return 6371000 * 2 * Math.asin(Math.min(1, Math.sqrt(h)));
};

const PORTAL_ACTION_RANGE_METERS = 40;
const AP_NOTICE_STAGGER_MS = 120;
const AP_NOTICE_ANIMATION_MS = 1000;
const AP_NOTICE_REMOVE_BUFFER_MS = 50;
const ATTACK_PULSE_TTL_MS = 1500;
const CHARGE_REQUEST_AMOUNT = 500;
const CHARGE_PER_XM = 1;
const RESONATOR_MAX_ENERGY = {
  1: 1000,
  2: 1500,
  3: 2000,
  4: 2500,
  5: 3000,
  6: 4000,
  7: 5000,
  8: 6000
};
const AUTHORITY_MODAL_KEYS = new Set(['inventory', 'portal', 'portalDeploy', 'portalModDeploy', 'attack']);

const isWsOpen = (client) => client && client.status() === WebSocket.OPEN;

const formatRequestError = (err, fallback = '操作失败') => {
  const code = Number(err?.data?.code ?? err?.code);
  const message =
    String(
      err?.data?.message ||
        err?.errmsg ||
        err?.message ||
        fallback
    ).trim() || fallback;
  if (Number.isFinite(code) && code > 0) {
    return `${message} (${code})`;
  }
  return message;
};

const toLogTimestamp = (value) => {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string' && value.trim()) {
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return Date.now();
};

const toLogTime = (timestamp) => {
  const ts = Number(timestamp);
  const date = new Date(Number.isFinite(ts) ? ts : Date.now());
  const hh = String(date.getHours()).padStart(2, '0');
  const mm = String(date.getMinutes()).padStart(2, '0');
  return `${hh}:${mm}`;
};

const normalizeLogEntry = (entry = {}) => {
  const timestamp = toLogTimestamp(entry.timestamp);
  return {
    id: entry.id || `log-${timestamp}-${Math.random().toString(16).slice(2)}`,
    time: entry.time || toLogTime(timestamp),
    type: entry.type || 'LOG',
    text: entry.text || '',
    message: entry.message || entry.text || '',
    playerId: entry.playerId || '',
    portalId: entry.portalId || '',
    faction: entry.faction || '',
    mu: Number(entry.mu || 0),
    timestamp
  };
};

const parseHackLootItem = (itemType) => {
  const raw = String(itemType || '').trim();
  const normalized = raw.toUpperCase();
  const matchLevel = normalized.match(/_L(\d+)$/);
  const level = matchLevel ? Number(matchLevel[1]) : null;

  if (normalized.startsWith('KEY:')) {
    return { itemType: raw, kind: 'KEY', icon: '🗝', name: 'Portal Key', level: null, sortRank: 0 };
  }
  if (normalized.startsWith('XMP_L')) {
    return { itemType: raw, kind: 'XMP', icon: '✶', name: 'XMP Burster', level, sortRank: 1 };
  }
  if (normalized.startsWith('RESO_L')) {
    return { itemType: raw, kind: 'RESO', icon: '⌬', name: 'Resonator', level, sortRank: 2 };
  }
  if (normalized === 'CUBE' || normalized.startsWith('CUBE_L')) {
    const cubeLevel = normalized === 'CUBE' ? 1 : level;
    return { itemType: raw, kind: 'CUBE', icon: '◈', name: 'Power Cube', level: cubeLevel, sortRank: 3 };
  }
  if (normalized.startsWith('US_L')) {
    return { itemType: raw, kind: 'US', icon: '✦', name: 'Ultra Strike', level, sortRank: 4 };
  }
  return {
    itemType: raw,
    kind: 'OTHER',
    icon: '◇',
    name: raw || 'Unknown',
    level: level || null,
    sortRank: 99
  };
};

const buildHackLootItems = (items = []) => {
  const counter = new Map();
  (Array.isArray(items) ? items : []).forEach((itemType) => {
    const key = String(itemType || '').trim();
    if (!key) return;
    counter.set(key, (counter.get(key) || 0) + 1);
  });

  return Array.from(counter.entries())
    .map(([itemType, count]) => {
      const parsed = parseHackLootItem(itemType);
      return {
        ...parsed,
        count: Number(count || 0)
      };
    })
    .filter((item) => item.count > 0)
    .sort((a, b) => {
      if (a.sortRank !== b.sortRank) return a.sortRank - b.sortRank;
      if ((a.level || 0) !== (b.level || 0)) return (b.level || 0) - (a.level || 0);
      return a.name.localeCompare(b.name);
    });
};

function AuthScreen({ mode, setMode, form, setForm, onSubmit, submitting, error }) {
  return (
    <div className="auth-screen">
      <div className="auth-card">
        <h2>OpenIngress RTS</h2>
        <p className="muted">请先登录后进入地图</p>
        <div className="auth-tabs">
          <button className={mode === 'login' ? 'active' : ''} onClick={() => setMode('login')}>
            登录
          </button>
          <button className={mode === 'register' ? 'active' : ''} onClick={() => setMode('register')}>
            注册
          </button>
        </div>

        <form
          className="auth-form"
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit();
          }}
        >
          <label>
            用户名
            <input
              value={form.username}
              onChange={(event) => setForm((prev) => ({ ...prev, username: event.target.value }))}
              required
            />
          </label>
          <label>
            密码
            <input
              type="password"
              value={form.password}
              onChange={(event) => setForm((prev) => ({ ...prev, password: event.target.value }))}
              required
            />
          </label>
          {mode === 'register' ? (
            <label>
              阵营
              <select
                value={form.faction}
                onChange={(event) => setForm((prev) => ({ ...prev, faction: event.target.value }))}
              >
                <option value="RESISTANCE">RESISTANCE</option>
                <option value="ENLIGHTENED">ENLIGHTENED</option>
              </select>
            </label>
          ) : null}

          {error ? <div className="auth-error">{error}</div> : null}
          <button className="primary auth-submit" type="submit" disabled={submitting}>
            {submitting ? '提交中...' : mode === 'login' ? '登录' : '注册'}
          </button>
        </form>
      </div>
    </div>
  );
}

export default function App() {
  const [authTokens, setAuthTokens] = useState(() =>
    authStore.get() || { accessToken: '', refreshToken: '', playerId: '' }
  );
  const [authReady, setAuthReady] = useState(false);
  const [authMode, setAuthMode] = useState('login');
  const [authForm, setAuthForm] = useState({ username: '', password: '', faction: 'RESISTANCE' });
  const [authSubmitting, setAuthSubmitting] = useState(false);
  const [authError, setAuthError] = useState('');

  const [openModal, setOpenModal] = useState(initialModals);
  const [selectedPortalId, setSelectedPortalId] = useState(null);
  const [player, setPlayer] = useState(null);
  const [inventoryMap, setInventoryMap] = useState({});
  const [portalsById, setPortalsById] = useState({});
  const [linksById, setLinksById] = useState({});
  const [fieldsById, setFieldsById] = useState({});
  const [nearbyPlayers, setNearbyPlayers] = useState([]);
  const [leaderboard, setLeaderboard] = useState([]);
  const [logs, setLogs] = useState([]);
  const [activePlayerProfile, setActivePlayerProfile] = useState(null);
  const [playerModalLoading, setPlayerModalLoading] = useState(false);
  const [playerModalError, setPlayerModalError] = useState('');
  const [toast, setToast] = useState('');
  const [mapBounds, setMapBounds] = useState(null);
  const [attackSpecs, setAttackSpecs] = useState(null);
  const [linkMode, setLinkMode] = useState({ active: false, fromPortalId: null });
  const [lootNoticeQueue, setLootNoticeQueue] = useState([]);
  const [activeLootNotice, setActiveLootNotice] = useState(null);
  const [activeApNotices, setActiveApNotices] = useState([]);
  const [attackPulses, setAttackPulses] = useState([]);
  const [attackChargeFx, setAttackChargeFx] = useState(null);

  const wsRef = useRef(null);
  const portalsRef = useRef({});
  const toastTimerRef = useRef(null);
  const boundsTimerRef = useRef(null);
  const lootNoticeTimerRef = useRef(null);
  const apNoticeTimersRef = useRef(new Set());
  const apNoticeActiveCountRef = useRef(0);
  const playerModalReqRef = useRef(0);
  const authorityRefreshRef = useRef(null);

  const showToast = useCallback((message) => {
    if (!message) return;
    if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
    setToast(message);
    toastTimerRef.current = setTimeout(() => setToast(''), 1800);
  }, []);

  const clearApNoticeTimers = useCallback(() => {
    apNoticeTimersRef.current.forEach((timerID) => clearTimeout(timerID));
    apNoticeTimersRef.current.clear();
    apNoticeActiveCountRef.current = 0;
  }, []);

  const markAuthInvalid = useCallback(() => {
    authStore.clear();
    setAuthTokens({ accessToken: '', refreshToken: '', playerId: '' });
    setAuthReady(false);
    setPlayer(null);
    setInventoryMap({});
    setPortalsById({});
    setLinksById({});
    setFieldsById({});
    setNearbyPlayers([]);
    setOpenModal(initialModals);
    setSelectedPortalId(null);
    setAttackSpecs(null);
    setActivePlayerProfile(null);
    setPlayerModalLoading(false);
    setPlayerModalError('');
    setLinkMode({ active: false, fromPortalId: null });
    setLootNoticeQueue([]);
    setActiveLootNotice(null);
    setActiveApNotices([]);
    setAttackPulses([]);
    setAttackChargeFx(null);
    authorityRefreshRef.current = null;
    if (lootNoticeTimerRef.current) {
      clearTimeout(lootNoticeTimerRef.current);
      lootNoticeTimerRef.current = null;
    }
    clearApNoticeTimers();
    if (wsRef.current) {
      wsRef.current.disconnect();
      wsRef.current = null;
    }
  }, [clearApNoticeTimers]);

  const http = useMemo(
    () =>
      createHttpClient({
        getAccessToken: () => authTokens.accessToken,
        getRefreshToken: () => authTokens.refreshToken,
        onAuthUpdate: (patch) => {
          setAuthTokens((prev) => {
            const next = { ...prev, ...patch };
            authStore.set(next);
            return next;
          });
        },
        onAuthInvalid: markAuthInvalid
      }),
    [authTokens.accessToken, authTokens.refreshToken, markAuthInvalid]
  );

  const portals = useMemo(() => Object.values(portalsById), [portalsById]);
  const links = useMemo(() => Object.values(linksById), [linksById]);
  const fields = useMemo(() => Object.values(fieldsById), [fieldsById]);
  const inventory = useMemo(() => mapInventory(inventoryMap, portalsById), [inventoryMap, portalsById]);
  const playerNameById = useMemo(() => {
    const next = {};
    if (player?.id) {
      next[player.id] = player.username || player.id;
    }
    (Array.isArray(leaderboard) ? leaderboard : []).forEach((item) => {
      const id = String(item?.id || item?.playerId || '');
      if (!id) return;
      next[id] = item?.username || item?.name || id;
    });
    return next;
  }, [leaderboard, player?.id, player?.username]);
  const portalNameById = useMemo(() => {
    const next = {};
    Object.values(portalsById || {}).forEach((portal) => {
      const id = String(portal?.id || '');
      if (!id) return;
      next[id] = portal?.title || portal?.name || id;
    });
    return next;
  }, [portalsById]);
  const portalIdByName = useMemo(() => {
    const next = {};
    Object.values(portalsById || {}).forEach((portal) => {
      const id = String(portal?.id || '');
      const name = String(portal?.title || portal?.name || '').trim();
      if (!id || !name) return;
      if (!next[name]) next[name] = [];
      next[name].push(id);
    });
    return next;
  }, [portalsById]);

  const selectedPortal = useMemo(() => {
    if (!selectedPortalId) return null;
    return portalsById[selectedPortalId] || null;
  }, [selectedPortalId, portalsById]);

  const hasPortalKey = useCallback(
    (portalId) => {
      if (!portalId) return false;
      return Number(inventoryMap?.[`KEY:${portalId}`] || 0) > 0;
    },
    [inventoryMap]
  );

  const portalDistanceMeters = useCallback(
    (portal) => {
      if (!player || !portal) return Number.POSITIVE_INFINITY;
      const me = {
        latitude: Number(player.renderLatitude ?? player.latitude),
        longitude: Number(player.renderLongitude ?? player.longitude)
      };
      const target = {
        latitude: Number(portal?.position?.latitude ?? portal?.lat),
        longitude: Number(portal?.position?.longitude ?? portal?.lng)
      };
      if (
        !Number.isFinite(me.latitude) ||
        !Number.isFinite(me.longitude) ||
        !Number.isFinite(target.latitude) ||
        !Number.isFinite(target.longitude)
      ) {
        return Number.POSITIVE_INFINITY;
      }
      return haversineMeters(me, target);
    },
    [player]
  );

  const isPortalInRange = useCallback(
    (portal) => portalDistanceMeters(portal) <= PORTAL_ACTION_RANGE_METERS,
    [portalDistanceMeters]
  );

  const selectedPortalInRange = useMemo(
    () => (selectedPortal ? isPortalInRange(selectedPortal) : false),
    [isPortalInRange, selectedPortal]
  );

  const selectedPortalChargeState = useMemo(() => {
    const fallback = {
      inRangeOrHasKey: false,
      isPortalFull: true,
      expectedApplied: 0,
      expectedXmCost: 0,
      hasEnoughXm: true,
      canCharge: false,
      disabledText: '未选择 Portal'
    };
    if (!selectedPortal?.id) return fallback;
    const inRangeOrHasKey = selectedPortalInRange || hasPortalKey(selectedPortal.id);
    const maxEnergy = (selectedPortal.resonators || []).reduce((sum, resonator) => {
      const level = Number(resonator?.level || 0);
      return sum + Number(RESONATOR_MAX_ENERGY[level] || 0);
    }, 0);
    const portalEnergy = Number(selectedPortal?.energy || 0);
    const remainingEnergy = Math.max(0, maxEnergy - portalEnergy);
    const isPortalFull = remainingEnergy <= 0;
    const expectedApplied = Math.min(CHARGE_REQUEST_AMOUNT, remainingEnergy);
    const expectedXmCost = Math.max(0, Math.ceil(expectedApplied * CHARGE_PER_XM));
    const playerXM = Number(player?.xm || 0);
    const hasEnoughXm = playerXM >= expectedXmCost;
    const canCharge = inRangeOrHasKey && !isPortalFull && hasEnoughXm;
    let disabledText = '';
    if (!inRangeOrHasKey) {
      disabledText = '超出40m且无Key';
    } else if (isPortalFull) {
      disabledText = 'Portal 已满电';
    } else if (!hasEnoughXm) {
      disabledText = `XM不足（需 ${expectedXmCost}）`;
    }
    return {
      inRangeOrHasKey,
      isPortalFull,
      expectedApplied,
      expectedXmCost,
      hasEnoughXm,
      canCharge,
      disabledText
    };
  }, [hasPortalKey, player?.xm, selectedPortal, selectedPortalInRange]);

  const hasOpenModal = useMemo(() => Object.values(openModal).some(Boolean), [openModal]);
  const hasBackdropModal = useMemo(
    () => Object.entries(openModal).some(([key, isOpen]) => Boolean(isOpen) && key !== 'attack'),
    [openModal]
  );

  useEffect(() => {
    portalsRef.current = portalsById;
  }, [portalsById]);

  const addLog = useCallback((entry) => {
    const nextEntry = normalizeLogEntry(entry);
    setLogs((prev) =>
      [...prev, nextEntry]
        .sort((a, b) => Number(a.timestamp || 0) - Number(b.timestamp || 0))
        .slice(-120)
    );
  }, []);

  useEffect(() => {
    if (activeLootNotice || !lootNoticeQueue.length) return;
    const [next, ...rest] = lootNoticeQueue;
    setLootNoticeQueue(rest);
    setActiveLootNotice(next);
  }, [activeLootNotice, lootNoticeQueue]);

  useEffect(() => {
    if (!activeLootNotice) return;
    if (lootNoticeTimerRef.current) clearTimeout(lootNoticeTimerRef.current);
    lootNoticeTimerRef.current = setTimeout(() => {
      setActiveLootNotice(null);
      lootNoticeTimerRef.current = null;
    }, 1000);
    return () => {
      if (lootNoticeTimerRef.current) {
        clearTimeout(lootNoticeTimerRef.current);
        lootNoticeTimerRef.current = null;
      }
    };
  }, [activeLootNotice]);

  const enqueueLootNotice = useCallback((payload) => {
    const items = Array.isArray(payload?.items) ? payload.items.filter(Boolean) : [];
    if (!items.length) return;
    const createdAt = Date.now();
    const notice = {
      id: payload?.id || `loot-${createdAt}-${Math.random().toString(16).slice(2)}`,
      portalId: payload?.portalId || '',
      portalTitle: payload?.portalTitle || payload?.portalId || 'Portal',
      items,
      createdAt
    };
    setLootNoticeQueue((prev) => [...prev, notice]);
  }, []);

  const enqueueApNotice = useCallback((payload) => {
    const ap = Number(payload?.ap || 0);
    if (!Number.isFinite(ap) || ap <= 0) return;
    const createdAt = Date.now();
    const delayMs = apNoticeActiveCountRef.current * AP_NOTICE_STAGGER_MS;
    const removeAfterMs = AP_NOTICE_ANIMATION_MS + delayMs + AP_NOTICE_REMOVE_BUFFER_MS;
    const noticeID = payload?.id || `ap-${createdAt}-${Math.random().toString(16).slice(2)}`;
    const notice = {
      id: noticeID,
      ap,
      text: payload?.text || `+${ap} AP`,
      createdAt,
      delayMs
    };
    apNoticeActiveCountRef.current += 1;
    setActiveApNotices((prev) => [...prev, notice]);

    const timerID = setTimeout(() => {
      apNoticeTimersRef.current.delete(timerID);
      apNoticeActiveCountRef.current = Math.max(0, apNoticeActiveCountRef.current - 1);
      setActiveApNotices((prev) => prev.filter((item) => item.id !== noticeID));
    }, removeAfterMs);
    apNoticeTimersRef.current.add(timerID);
  }, []);

  const dismissLootNotice = useCallback(() => {
    if (lootNoticeTimerRef.current) {
      clearTimeout(lootNoticeTimerRef.current);
      lootNoticeTimerRef.current = null;
    }
    setActiveLootNotice(null);
  }, []);

  const setAuthFromResponse = useCallback((data) => {
    const next = {
      accessToken: data?.access_token || '',
      refreshToken: data?.refresh_token || '',
      playerId: data?.player_id || ''
    };
    authStore.set(next);
    setAuthTokens(next);
    setAuthReady(true);
  }, []);

  const fetchPlayer = useCallback(async () => {
    const me = await http.get('/api/v1/players/me');
    const mapped = mapPlayer(me);
    setPlayer(mapped);
    return mapped;
  }, [http]);

  const fetchInventory = useCallback(async () => {
    const data = await http.get('/api/v1/inventory');
    setInventoryMap(data || {});
  }, [http]);

  const refreshAuthoritySnapshot = useCallback(async () => {
    if (authorityRefreshRef.current) return authorityRefreshRef.current;
    const job = Promise.allSettled([fetchPlayer(), fetchInventory()]).finally(() => {
      authorityRefreshRef.current = null;
    });
    authorityRefreshRef.current = job;
    return job;
  }, [fetchInventory, fetchPlayer]);

  const applyResourceSnapshot = useCallback(
    (payload) => {
      if (!payload || typeof payload !== 'object' || Array.isArray(payload)) return false;

      let applied = false;
      const inventory = payload?.inventory;
      if (inventory && typeof inventory === 'object' && !Array.isArray(inventory)) {
        setInventoryMap(inventory);
        applied = true;
      }
      const inventoryDelta = payload?.inventoryDelta;
      if (inventoryDelta && typeof inventoryDelta === 'object' && !Array.isArray(inventoryDelta)) {
        setInventoryMap((prev) => {
          const next = { ...(prev || {}) };
          Object.entries(inventoryDelta).forEach(([itemID, diffRaw]) => {
            const diff = Number(diffRaw || 0);
            if (!itemID || !Number.isFinite(diff) || diff === 0) return;
            const current = Number(next[itemID] || 0);
            const value = Math.max(0, current + diff);
            if (value === 0) {
              delete next[itemID];
            } else {
              next[itemID] = value;
            }
          });
          return next;
        });
        applied = true;
      }

      const source = payload?.player && typeof payload.player === 'object' ? payload.player : payload;
      const playerID = String(source?.playerId || source?.id || '');
      if (playerID && authTokens.playerId && playerID !== authTokens.playerId) {
        return applied;
      }

      const nextLevel = Number(source?.level);
      const nextAP = Number(source?.ap);
      const nextXM = Number(source?.xm);
      const nextMaxXM = Number(source?.maxXm);
      const apGained = Number(source?.apGained || 0);
      const xmDelta = Number(source?.xmDelta || 0);
      const hasPlayerFields =
        Number.isFinite(nextLevel) ||
        Number.isFinite(nextAP) ||
        Number.isFinite(nextXM) ||
        Number.isFinite(nextMaxXM) ||
        Number.isFinite(apGained) ||
        Number.isFinite(xmDelta);
      if (!hasPlayerFields) {
        return applied;
      }

      setPlayer((prev) => {
        if (!prev) return prev;
        const calcAP = Number.isFinite(nextAP)
          ? nextAP
          : prev.ap + (Number.isFinite(apGained) ? apGained : 0);
        const rawXM = Number.isFinite(nextXM)
          ? nextXM
          : prev.xm + (Number.isFinite(xmDelta) ? xmDelta : 0);
        const calcMaxXM = Number.isFinite(nextMaxXM) ? nextMaxXM : prev.maxXm;
        return {
          ...prev,
          level: Number.isFinite(nextLevel) ? nextLevel : prev.level,
          ap: calcAP,
          xm: Math.max(0, Math.min(calcMaxXM, rawXM)),
          maxXm: calcMaxXM
        };
      });
      return true;
    },
    [authTokens.playerId]
  );

  const fetchLeaderboard = useCallback(async () => {
    try {
      const list = await http.get('/api/v1/leaderboard', { auth: false });
      setLeaderboard(Array.isArray(list) ? list : []);
    } catch (_err) {
      setLeaderboard([]);
    }
  }, [http]);

  const fetchLogs = useCallback(async () => {
    try {
      const data = await http.get('/api/v1/logs/global', { auth: false });
      const rows = (data?.logs || [])
        .map((log) =>
          normalizeLogEntry({
            id: log.id || `${log.type}-${log.timestamp || Date.now()}`,
            type: log.type || 'LOG',
            text: log.message || '',
            message: log.message || '',
            playerId: log.playerId || '',
            portalId: log.portalId || '',
            faction: log.faction || '',
            mu: Number(log.mu || 0),
            timestamp: log.timestamp
          })
        )
        .sort((a, b) => Number(a.timestamp || 0) - Number(b.timestamp || 0));
      setLogs(rows.slice(-120));
    } catch (_err) {
      setLogs([]);
    }
  }, [http]);

  const fetchRuntimeConfig = useCallback(async () => {
    try {
      const data = await http.get('/api/v1/config', { auth: false });
      setAttackSpecs(normalizeAttackSpecs(data?.attackSpecs));
    } catch (_err) {
      setAttackSpecs(null);
    }
  }, [http]);

  const fetchMapEntities = useCallback(
    async (bounds) => {
      if (!bounds) return;
      const query = new URLSearchParams({
        minLat: String(bounds.minLat),
        maxLat: String(bounds.maxLat),
        minLon: String(bounds.minLon),
        maxLon: String(bounds.maxLon)
      });
      const data = await http.get(`/api/v1/map/entities?${query.toString()}`);
      const incomingPortals = Object.fromEntries((data?.portals || []).map((p) => {
        const mapped = mapPortalEntity(p);
        return [mapped.id, mapped];
      }));
      const incomingLinks = Object.fromEntries((data?.links || []).map((item) => {
        const mapped = mapLinkEntity(item);
        return [mapped.id, mapped];
      }));
      const incomingFields = Object.fromEntries((data?.fields || []).map((item) => {
        const mapped = mapFieldEntity(item, incomingPortals);
        return [mapped.id, mapped];
      }));

      setPortalsById(incomingPortals);
      setLinksById(incomingLinks);
      setFieldsById(incomingFields);
    },
    [http]
  );

  const handleWsMessage = useCallback(
    (message) => {
      const type = message?.type;
      const data = message?.data || {};
      const apGained = Number(data.apGained || 0);
      const rewardPlayerID = String(data.playerId || '');
      const shouldHandleAPNotice =
        type === 'PLAYER_RESOURCE_UPDATE' ||
        type === 'HACK_RESULT' ||
        type === 'AUTO_HACK_RESULT';
      if (
        shouldHandleAPNotice &&
        Number.isFinite(apGained) &&
        apGained > 0 &&
        (!rewardPlayerID || rewardPlayerID === authTokens.playerId)
      ) {
        enqueueApNotice({ ap: apGained, text: `+${apGained} AP` });
      }

      if (type === 'PLAYER_STATE') {
        setPlayer((prev) => {
          if (!prev) return prev;
          const latitude = Number(data.latitude ?? prev.latitude);
          const longitude = Number(data.longitude ?? prev.longitude);
          const renderLatitude = Number(data.renderLatitude ?? latitude);
          const renderLongitude = Number(data.renderLongitude ?? longitude);
          const preRenderLatitude = Number(data.preRenderLatitude ?? renderLatitude);
          const preRenderLongitude = Number(data.preRenderLongitude ?? renderLongitude);
          const nextSpeed = Number(data.speedMps);
          const nextHeading = Number(data.headingDeg);
          const incomingTarget = parsePlayerTarget(data);
          let nextTarget = incomingTarget || prev.target || null;
          if (
            nextTarget &&
            haversineMeters(
              { latitude: renderLatitude, longitude: renderLongitude },
              nextTarget
            ) <= 4
          ) {
            nextTarget = null;
          }
          return {
            ...prev,
            latitude,
            longitude,
            renderLatitude,
            renderLongitude,
            preRenderLatitude,
            preRenderLongitude,
            speedMps: Number.isFinite(nextSpeed) ? nextSpeed : 0,
            headingDeg: Number.isFinite(nextHeading) ? nextHeading : prev.headingDeg,
            target: nextTarget,
            targetLatitude: nextTarget?.latitude ?? null,
            targetLongitude: nextTarget?.longitude ?? null
          };
        });
        return;
      }

      if (type === 'NEARBY_PLAYERS') {
        setNearbyPlayers(
          (data.players || []).map((item) => ({
            id: item.id,
            latitude: Number(item.latitude),
            longitude: Number(item.longitude),
            renderLatitude: Number(item.renderLatitude ?? item.latitude),
            renderLongitude: Number(item.renderLongitude ?? item.longitude),
            preRenderLatitude: Number(item.preRenderLatitude ?? item.renderLatitude ?? item.latitude),
            preRenderLongitude: Number(item.preRenderLongitude ?? item.renderLongitude ?? item.longitude)
          }))
        );
        return;
      }

      if (type === 'MAP_UPDATE' || type === 'MAP_TICK') {
        const full = Boolean(data.full);
        const mapped = (data.portals || []).map((item) => {
          const lat = Number(item.latitude ?? item.lat);
          const lng = Number(item.longitude ?? item.lng);
          const title = item.title || item.name || item.id || 'Portal';
          const coverURL = item.cover_url || item.coverUrl || item.image || '';
          return {
            id: item.id,
            title,
            name: title,
            cover_url: coverURL,
            image: coverURL,
            faction: item.faction || 'NEUTRAL',
            level: Number(item.level || 1),
            energy: Number(item.energy || 0),
            lat: Number.isFinite(lat) ? lat : 0,
            lng: Number.isFinite(lng) ? lng : 0,
            position: {
              latitude: Number.isFinite(lat) ? lat : 0,
              longitude: Number.isFinite(lng) ? lng : 0
            }
          };
        });

        if (full) {
          setPortalsById((prev) => {
            const next = {};
            mapped.forEach((portal) => {
              const current = prev[portal.id];
              next[portal.id] = current ? { ...current, ...portal } : portal;
            });
            return next;
          });
        } else {
          setPortalsById((prev) => {
            const next = { ...prev };
            mapped.forEach((portal) => {
              const current = prev[portal.id];
              next[portal.id] = current ? { ...current, ...portal } : portal;
            });
            return next;
          });
        }
        return;
      }

      if (type === 'PORTAL_UPDATE') {
        const patch = portalPatchFromUpdate(data);
        setPortalsById((prev) => {
          const id = patch.id;
          if (!id || !prev[id]) return prev;
          return {
            ...prev,
            [id]: applyPortalPatch(prev[id], patch)
          };
        });
        return;
      }

      if (type === 'PLAYER_RESOURCE_UPDATE') {
        applyResourceSnapshot(data);
        return;
      }

      if (type === 'LINK_UPDATE') {
        const mapped = mapLinkEntity(data);
        setLinksById((prev) => ({ ...prev, [mapped.id]: mapped }));
        addLog(formatWsLog('LINK', message));
        return;
      }

      if (type === 'FIELD_CREATED') {
        const mapped = mapFieldEntity(data, portalsRef.current);
        if (mapped.id) {
          setFieldsById((prev) => ({ ...prev, [mapped.id]: mapped }));
        }
        addLog(formatWsLog('FIELD', message));
        return;
      }

      if (type === 'ATTACK_RESULT') {
        addLog(formatWsLog('ATTACK', message));
        return;
      }

      if (type === 'AUTO_HACK_RESULT') {
        if (data.success) {
          const items = Array.isArray(data.itemsGained) ? data.itemsGained.filter(Boolean) : [];
          if (items.length) {
            const portalID = String(data.portalId || '');
            const portalEntity = portalID ? portalsRef.current?.[portalID] : null;
            const portalTitle = portalEntity?.title || portalEntity?.name || portalID || 'Portal';
            enqueueLootNotice({
              portalId: portalID,
              portalTitle,
              items: buildHackLootItems(items)
            });
          }
        }
        return;
      }

      if (type === 'ERROR') {
        showToast(formatRequestError({ data }, '操作失败'));
        return;
      }

      if (type && type !== 'PONG' && type !== 'CONNECTED' && type !== 'AUTO_HACK_RESULT') {
        addLog(formatWsLog(type, message));
      }
    },
    [addLog, applyResourceSnapshot, authTokens.playerId, enqueueApNotice, enqueueLootNotice, showToast]
  );

  const connectWs = useCallback(() => {
    if (!authTokens.accessToken || wsRef.current) return;
    const client = createWsClient({
      url: runtime.wsUrl,
      getAuthToken: () => authTokens.accessToken,
      onOpen: () => {
        addLog({ id: `ws-open-${Date.now()}`, time: new Date().toTimeString().slice(0, 5), type: 'WS', text: 'connected' });
      },
      onClose: () => {
        addLog({ id: `ws-close-${Date.now()}`, time: new Date().toTimeString().slice(0, 5), type: 'WS', text: 'disconnected' });
      },
      onMessage: handleWsMessage,
      onError: () => {
        showToast('WS 连接异常');
      }
    });
    wsRef.current = client;
    client.connect();
  }, [addLog, authTokens.accessToken, handleWsMessage, showToast]);

  const disconnectWs = useCallback(() => {
    if (!wsRef.current) return;
    wsRef.current.disconnect();
    wsRef.current = null;
  }, []);

  useEffect(() => {
    const bootstrap = async () => {
      const stored = authStore.get();
      if (!stored?.accessToken) {
        setAuthReady(false);
        return;
      }
      setAuthTokens(stored);
      setAuthReady(true);
    };
    bootstrap();

    return () => {
      if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
      if (boundsTimerRef.current) clearTimeout(boundsTimerRef.current);
      if (lootNoticeTimerRef.current) clearTimeout(lootNoticeTimerRef.current);
      clearApNoticeTimers();
      disconnectWs();
    };
  }, [clearApNoticeTimers, disconnectWs]);

  useEffect(() => {
    if (!authReady) return;
    connectWs();
    fetchPlayer()
      .then((me) => {
        const bounds = defaultBounds(me);
        setMapBounds(bounds);
        fetchMapEntities(bounds).catch(() => {});
      })
      .catch((err) => {
        showToast(err?.errmsg || '玩家信息加载失败');
      });
    fetchLeaderboard().catch(() => {});
    fetchLogs().catch(() => {});
    fetchRuntimeConfig().catch(() => {});
  }, [authReady, connectWs, fetchLeaderboard, fetchLogs, fetchMapEntities, fetchPlayer, fetchRuntimeConfig, showToast]);

  useEffect(() => {
    if (!authReady) {
      disconnectWs();
    }
  }, [authReady, disconnectWs]);

  const open = (key) => {
    if (key === 'player') {
      setActivePlayerProfile(player || null);
      setPlayerModalLoading(false);
      setPlayerModalError('');
    }
    if (AUTHORITY_MODAL_KEYS.has(key)) {
      refreshAuthoritySnapshot().catch(() => {});
    }
    setOpenModal((prev) => ({ ...prev, [key]: true }));
  };
  const close = (key) => setOpenModal((prev) => ({ ...prev, [key]: false }));
  const closeAll = () => setOpenModal(initialModals);

  const handleLogPlayerClick = useCallback(
    async (playerId) => {
      const targetID = String(playerId || '').trim();
      if (!targetID) return;
      const cached =
        (player?.id === targetID ? player : null) ||
        mapPlayer((leaderboard || []).find((item) => String(item?.id || item?.playerId || '') === targetID));
      setActivePlayerProfile(cached || null);
      setPlayerModalError('');
      setPlayerModalLoading(true);
      setOpenModal((prev) => ({ ...prev, player: true }));

      const reqID = playerModalReqRef.current + 1;
      playerModalReqRef.current = reqID;
      try {
        const data = await http.get(`/api/v1/players/${encodeURIComponent(targetID)}`);
        if (playerModalReqRef.current !== reqID) return;
        setActivePlayerProfile(mapPlayer(data));
      } catch (err) {
        if (playerModalReqRef.current !== reqID) return;
        setPlayerModalError(err?.errmsg || '玩家信息加载失败');
        showToast(err?.errmsg || '玩家信息加载失败');
      } finally {
        if (playerModalReqRef.current === reqID) {
          setPlayerModalLoading(false);
        }
      }
    },
    [http, leaderboard, player, showToast]
  );

  const handleLogPortalClick = useCallback(
    async (portalId) => {
      const targetID = String(portalId || '').trim();
      if (!targetID) return;
      if (portalsById[targetID]) {
        setSelectedPortalId(targetID);
        setOpenModal((prev) => ({ ...prev, portal: true }));
        return;
      }
      try {
        const data = await http.get(`/api/v1/portals/${encodeURIComponent(targetID)}`);
        const mapped = mapPortalEntity(data);
        setPortalsById((prev) => ({ ...prev, [mapped.id]: mapped }));
        setSelectedPortalId(mapped.id);
        setOpenModal((prev) => ({ ...prev, portal: true }));
      } catch (err) {
        showToast(err?.errmsg || 'Portal 信息加载失败');
      }
    },
    [http, portalsById, showToast]
  );

  const handleAuthSubmit = async () => {
    setAuthSubmitting(true);
    setAuthError('');
    try {
      const endpoint = authMode === 'login' ? '/api/v1/auth/login' : '/api/v1/auth/register';
      const payload =
        authMode === 'login'
          ? { username: authForm.username, password: authForm.password }
          : {
              username: authForm.username,
              password: authForm.password,
              faction: authForm.faction
            };
      const data = await http.post(endpoint, payload, { auth: false });
      setAuthFromResponse(data);
    } catch (err) {
      setAuthError(err?.errmsg || '鉴权失败');
    } finally {
      setAuthSubmitting(false);
    }
  };

  const handleLogout = useCallback(() => {
    markAuthInvalid();
    setActivePlayerProfile(null);
    setPlayerModalLoading(false);
    setPlayerModalError('');
    setAuthMode('login');
    setAuthForm({ username: '', password: '', faction: 'RESISTANCE' });
    setAuthError('');
  }, [markAuthInvalid]);

  const handleMoveTarget = useCallback(
    ({ latitude, longitude }) => {
      const target = { latitude: Number(latitude), longitude: Number(longitude) };
      setPlayer((prev) =>
        prev
          ? {
              ...prev,
              target,
              targetLatitude: target.latitude,
              targetLongitude: target.longitude
            }
          : prev
      );
      if (!wsRef.current || !isWsOpen(wsRef.current)) return;
      wsRef.current.send('PLAYER_TARGET_UPDATE', { latitude, longitude });
    },
    []
  );

  const handleViewUpdate = useCallback(
    (bounds) => {
      setMapBounds(bounds);
      if (boundsTimerRef.current) clearTimeout(boundsTimerRef.current);
      boundsTimerRef.current = setTimeout(() => {
        fetchMapEntities(bounds).catch(() => {});
        if (wsRef.current && isWsOpen(wsRef.current)) {
          wsRef.current.send('PLAYER_VIEW_UPDATE', { bounds });
        }
      }, 220);
    },
    [fetchMapEntities]
  );

  const sendResonatorDeploy = async (portalId, slot, level, expectedVersion = 0) => {
    const client = wsRef.current;
    if (!client || !isWsOpen(client)) {
      showToast('WS 未连接');
      return;
    }
    try {
      await client.request(
        'PLAYER_DEPLOY_RESONATOR',
        { portalId, slot, level, expectedVersion },
        1200
      );
    } catch (err) {
      showToast(err?.data?.message || err?.message || '部署失败');
    }
  };

  const sendModDeploy = async (portalId, slot, mod) => {
    const client = wsRef.current;
    if (!client || !isWsOpen(client)) {
      showToast('WS 未连接');
      return;
    }
    const normalizeModType = () => {
      const raw = String(mod?.modType || mod?.subtype || mod?.type || '').toUpperCase();
      if (raw === 'LINK') return 'LINK_AMP';
      if (raw === 'HEATSINK') return 'HEAT_SINK';
      if (raw === 'MULTI') return 'MULTI_HACK';
      if (raw === 'FORCE') return 'FORCE_AMP';
      if (raw === 'PORTAL_SHIELD') return 'SHIELD';
      if (raw === 'SHIELD') return 'SHIELD';
      if (raw === 'SBUL') return 'SOFTBANK_ULTRA_LINK';
      return raw;
    };

    try {
      await client.request(
        'PLAYER_DEPLOY_MOD',
        {
          portalId,
          slot,
          modType: normalizeModType(),
          rarity: mod.rarity || 'C',
          expectedVersion: 0
        },
        1200
      );
    } catch (err) {
      showToast(err?.data?.message || err?.message || 'Mod 部署失败');
    }
  };

  const handleUpdateResonators = async (portalId, nextSlots) => {
    const current = portalsById[portalId];
    if (!current) return;
    if (!isPortalInRange(current)) {
      showToast('超出40m，无法 Deploy');
      return;
    }
    const prevSlots = current.resonators || [];
    const idx = nextSlots.findIndex((slot, i) => {
      const prev = prevSlots[i];
      return (slot?.level || 0) !== (prev?.level || 0);
    });
    if (idx < 0) return;
    const next = nextSlots[idx];
    if (!next?.level) return;
    const prev = prevSlots[idx] || null;
    const expectedVersion = Number(prev?.version || 0);
    sendResonatorDeploy(portalId, idx + 1, Number(next.level || 1), expectedVersion);
  };

  const handleUpdateMods = async (portalId, nextSlots) => {
    const current = portalsById[portalId];
    if (!current) return;
    if (!isPortalInRange(current)) {
      showToast('超出40m，无法 Install Mod');
      return;
    }
    const prevSlots = current.mods || [];
    const idx = nextSlots.findIndex((slot, i) => {
      const prev = prevSlots[i];
      return (slot?.subtype || '') !== (prev?.subtype || '');
    });
    if (idx < 0) return;
    const next = nextSlots[idx];
    if (!next?.subtype) return;
    sendModDeploy(portalId, idx + 1, next);
  };

  const recycleItems = async (ids) => {
    if (!ids?.length) return;
    try {
      for (const id of ids) {
        const item = inventory.find((it) => it.id === id);
        if (!item || item.count <= 0) continue;
        await http.post('/api/v1/inventory/recycle', {
          itemType: item.id,
          amount: item.count
        });
      }
      await fetchInventory();
      showToast('回收完成');
    } catch (err) {
      showToast(err?.errmsg || '回收失败');
    }
  };

  const useItem = async (itemId, amount = 1) => {
    if (!itemId || amount <= 0) return false;
    try {
      const data = await http.post('/api/v1/inventory/use', {
        itemType: itemId,
        amount
      });
      applyResourceSnapshot(data);
      setInventoryMap((prev) => {
        const next = { ...(prev || {}) };
        const current = Number(next[itemId] || 0);
        const remain = Math.max(0, current - Number(amount || 0));
        if (remain === 0) {
          delete next[itemId];
        } else {
          next[itemId] = remain;
        }
        return next;
      });
      return true;
    } catch (err) {
      showToast(err?.errmsg || '道具使用失败');
      return false;
    }
  };

  const emitAttackPulse = useCallback(
    ({ weaponType, weaponLevel, chargeBonus }) => {
      const latitude = Number(player?.renderLatitude ?? player?.latitude);
      const longitude = Number(player?.renderLongitude ?? player?.longitude);
      if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return;
      const baseRadius = attackRadiusForWeapon(weaponType, weaponLevel, attackSpecs);
      if (!baseRadius || baseRadius <= 0) return;
      const clampedCharge = Math.max(0, Math.min(0.2, Number(chargeBonus || 0)));
      const radius = baseRadius * (1 + clampedCharge);
      const pulse = {
        id: `pulse-${Date.now()}-${Math.random().toString(16).slice(2)}`,
        latitude,
        longitude,
        radius,
        createdAt: Date.now()
      };
      setAttackPulses((prev) => {
        const now = Date.now();
        const kept = prev.filter((item) => now - Number(item.createdAt || 0) <= ATTACK_PULSE_TTL_MS);
        return [...kept, pulse].slice(-8);
      });
    },
    [attackSpecs, player?.latitude, player?.longitude, player?.renderLatitude, player?.renderLongitude]
  );

  useEffect(() => {
    if (!attackPulses.length) return undefined;
    const timer = window.setInterval(() => {
      const now = Date.now();
      setAttackPulses((prev) =>
        prev.filter((item) => now - Number(item.createdAt || 0) <= ATTACK_PULSE_TTL_MS)
      );
    }, 200);
    return () => window.clearInterval(timer);
  }, [attackPulses.length]);

  const handleAttackChargeFxChange = useCallback((value) => {
    if (!value || !value.active) {
      setAttackChargeFx(null);
      return;
    }
    setAttackChargeFx({
      active: true,
      startAt: Number(value.startAt || Date.now()),
      durationMs: Math.max(1, Number(value.durationMs || 1)),
      weaponLevel: Math.max(1, Math.min(8, Number(value.weaponLevel || 1)))
    });
  }, []);

  const sendAttack = async ({ weaponType, weaponLevel, chargeBonus }) => {
    const client = wsRef.current;
    if (!client || !isWsOpen(client)) {
      throw new Error('WS 未连接');
    }
    let response;
    try {
      response = await client.request(
        'PLAYER_ATTACK',
        {
          weaponType,
          weaponLevel,
          chargeBonus
        },
        1200
      );
    } catch (err) {
      throw new Error(formatRequestError(err, '攻击失败'));
    }
    if (response?.type === 'ERROR') {
      throw new Error(formatRequestError(response, '攻击失败'));
    }
    if (response?.type === 'TIMEOUT') {
      throw new Error('攻击请求超时');
    }
    return response?.data || {};
  };

  const handleAttack = async ({ item, charge }) => {
    const weapon = weaponForItem(item);
    if (!weapon) {
      showToast('仅支持 XMP/US');
      return { ok: false };
    }

    const chargeBonus = Math.max(0, Math.min(0.2, Number(charge || 0)));

    try {
      const attackData = await sendAttack({
        weaponType: weapon.weaponType,
        weaponLevel: weapon.weaponLevel,
        chargeBonus
      });
      emitAttackPulse({
        weaponType: weapon.weaponType,
        weaponLevel: weapon.weaponLevel,
        chargeBonus
      });
      const portalDamages = Array.isArray(attackData?.portalDamages) ? attackData.portalDamages : [];
      const affected = portalDamages.length;
      showToast(`已发射，命中 ${affected} 个目标`);
      return { ok: true, count: affected };
    } catch (err) {
      const message = formatRequestError(err, '攻击失败');
      showToast(message);
      return { ok: false, error: message };
    }
  };

  const handlePortalClick = useCallback(
    (portal) => {
      if (linkMode.active && linkMode.fromPortalId) {
        if (portal.id === linkMode.fromPortalId) return;
        const fromPortal = portalsById[linkMode.fromPortalId];
        if (!isPortalInRange(fromPortal)) {
          showToast('超出40m，无法 Link');
          return;
        }
        if (!isPortalInRange(portal)) {
          showToast('目标超出40m，无法 Link');
          return;
        }
        const client = wsRef.current;
        if (!client || !isWsOpen(client)) {
          showToast('WS 未连接');
          return;
        }
        client
          .request(
            'PLAYER_CREATE_LINK',
            {
              fromPortalId: linkMode.fromPortalId,
              toPortalId: portal.id
            },
            1000
          )
          .then(() => {
            setLinkMode({ active: false, fromPortalId: null });
            showToast('Link 已发送');
          })
          .catch((err) => {
            showToast(err?.data?.message || err?.message || 'Link 失败');
            setLinkMode({ active: false, fromPortalId: null });
          });
        return;
      }

      setSelectedPortalId(portal.id);
      open('portal');
    },
    [isPortalInRange, linkMode, portalsById, showToast]
  );

  const handleCancelLinkMode = useCallback(() => {
    setLinkMode({ active: false, fromPortalId: null });
    showToast('已退出 Link 模式');
  }, [showToast]);

  const handleChargePortal = () => {
    const portalId = selectedPortal?.id;
    if (!portalId) return;
    if (!selectedPortalChargeState.canCharge) {
      showToast(selectedPortalChargeState.disabledText || '当前无法 Charge');
      return;
    }
    const client = wsRef.current;
    if (!client || !isWsOpen(client)) {
      showToast('WS 未连接');
      return;
    }
    client
      .request('PLAYER_CHARGE_PORTAL', { portalId, amount: CHARGE_REQUEST_AMOUNT }, 1000)
      .then(() => showToast('充能请求已发送'))
      .catch((err) => showToast(err?.data?.message || err?.message || '充能失败'));
  };

  const handleToggleAutoHack = async (enabled) => {
    try {
      const updated = await http.patch('/api/v1/players/me', { autoHack: enabled });
      setPlayer(mapPlayer(updated));
      if (wsRef.current && isWsOpen(wsRef.current)) {
        wsRef.current.send('PLAYER_TOGGLE_AUTO_HACK', { enabled });
      }
    } catch (err) {
      showToast(err?.errmsg || '更新自动 Hack 失败');
    }
  };

  if (!authReady || !authTokens.accessToken) {
    return (
      <AuthScreen
        mode={authMode}
        setMode={setAuthMode}
        form={authForm}
        setForm={setAuthForm}
        onSubmit={handleAuthSubmit}
        submitting={authSubmitting}
        error={authError}
      />
    );
  }

  const playerAp = apProgress(player?.ap || 0, player?.level || 1);
  const modalPlayer = activePlayerProfile || player;
  const modalPlayerAp = apProgress(modalPlayer?.ap || 0, modalPlayer?.level || 1);

  return (
    <div className="screen">
      <TopBar onOpen={open} onLogout={handleLogout} player={player} ap={playerAp} />
      <NotifyLayer
        lootNotice={activeLootNotice}
        onCloseLoot={dismissLootNotice}
        apNotices={activeApNotices}
      />

      <main className="map-shell">
        <MapShell
          portals={portals}
          links={links}
          fields={fields}
          player={player}
          moveTarget={player?.target || null}
          attackPulses={attackPulses}
          attackChargeFx={attackChargeFx}
          nearbyPlayers={nearbyPlayers}
          onOpenPortal={handlePortalClick}
          onTargetUpdate={handleMoveTarget}
          onViewUpdate={handleViewUpdate}
          linkMode={linkMode}
          onCancelLinkMode={handleCancelLinkMode}
        />
        <LogDock
          logs={logs}
          compact={hasOpenModal}
          playerNameById={playerNameById}
          portalNameById={portalNameById}
          portalIdByName={portalIdByName}
          onPlayerClick={handleLogPlayerClick}
          onPortalClick={handleLogPortalClick}
        />
      </main>

      <PortalModal
        open={openModal.portal}
        onClose={() => close('portal')}
        portal={selectedPortal}
        playerName={player?.username || 'Agent'}
        playerId={player?.id || ''}
        onDeploy={() => {
          if (!selectedPortalInRange) {
            showToast('超出40m，无法 Deploy');
            return;
          }
          close('portal');
          open('portalDeploy');
        }}
        onModDeploy={() => {
          if (!selectedPortalInRange) {
            showToast('超出40m，无法 Install Mod');
            return;
          }
          close('portal');
          open('portalModDeploy');
        }}
        onLink={() => {
          if (!selectedPortal?.id) return;
          if (!selectedPortalInRange) {
            showToast('超出40m，无法 Link');
            return;
          }
          setLinkMode({ active: true, fromPortalId: selectedPortal.id });
          close('portal');
          showToast('Link 模式：请点击目标 Portal');
        }}
        canDeploy={selectedPortalInRange}
        canModDeploy={selectedPortalInRange}
        canLink={selectedPortalInRange}
        canCharge={selectedPortalChargeState.canCharge}
        deployDisabledText="超出40m"
        modDeployDisabledText="超出40m"
        linkDisabledText="超出40m"
        chargeDisabledText={selectedPortalChargeState.disabledText}
        onCharge={handleChargePortal}
      />

      <PortalDeployModal
        open={openModal.portalDeploy}
        onClose={() => {
          close('portalDeploy');
          open('portal');
        }}
        portal={selectedPortal}
        slots={selectedPortal?.resonators}
        onUpdateSlots={handleUpdateResonators}
        items={inventory}
        playerName={player?.username || 'Agent'}
        playerId={player?.id || ''}
        inRange={selectedPortalInRange}
      />

      <PortalModDeployModal
        open={openModal.portalModDeploy}
        onClose={() => {
          close('portalModDeploy');
          open('portal');
        }}
        portal={selectedPortal}
        modSlots={selectedPortal?.mods}
        onUpdateSlots={handleUpdateMods}
        items={inventory}
        playerName={player?.username || 'Agent'}
        playerId={player?.id || ''}
        inRange={selectedPortalInRange}
        playerFaction={player?.faction}
      />

      <PlayerModal
        open={openModal.player}
        onClose={() => close('player')}
        player={modalPlayer}
        ap={modalPlayerAp}
        title={modalPlayer?.id === player?.id ? 'Agent Profile' : 'Player Profile'}
        loading={playerModalLoading}
        error={playerModalError}
      />

      <InventoryModal
        open={openModal.inventory}
        onClose={() => close('inventory')}
        items={inventory}
        onRecycle={recycleItems}
        onUse={(itemId) => useItem(itemId, 1)}
      />

      <AttackModal
        open={openModal.attack}
        onClose={() => close('attack')}
        items={inventory}
        playerXm={player?.xm || 0}
        attackSpecs={attackSpecs}
        onAttack={handleAttack}
        onChargeFxChange={handleAttackChargeFxChange}
      />

      <LeaderboardModal open={openModal.leaderboard} onClose={() => close('leaderboard')} rows={leaderboard} />

      <SettingsModal
        open={openModal.settings}
        onClose={() => close('settings')}
        autoHack={Boolean(player?.autoHack)}
        onToggleAutoHack={handleToggleAutoHack}
      />

      <LayersModal open={openModal.layers} onClose={() => close('layers')} />

      {toast ? <div className="toast toast-global">{toast}</div> : null}

      {hasBackdropModal && (
        <div className="modal-backdrop" onClick={closeAll} />
      )}
    </div>
  );
}
