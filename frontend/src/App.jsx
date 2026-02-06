import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import TopBar from './components/TopBar.jsx';
import MapShell from './components/MapShell.jsx';
import LogDock from './components/LogDock.jsx';
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
import { attackCandidates, attackItemType, weaponForItem } from './constants/attackProfiles.js';

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

const isWsOpen = (client) => client && client.status() === WebSocket.OPEN;

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
  const [toast, setToast] = useState('');
  const [mapBounds, setMapBounds] = useState(null);
  const [linkMode, setLinkMode] = useState({ active: false, fromPortalId: null });

  const wsRef = useRef(null);
  const portalsRef = useRef({});
  const toastTimerRef = useRef(null);
  const boundsTimerRef = useRef(null);

  const showToast = useCallback((message) => {
    if (!message) return;
    if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
    setToast(message);
    toastTimerRef.current = setTimeout(() => setToast(''), 1800);
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
    setLinkMode({ active: false, fromPortalId: null });
    if (wsRef.current) {
      wsRef.current.disconnect();
      wsRef.current = null;
    }
  }, []);

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

  const selectedPortal = useMemo(() => {
    if (!selectedPortalId) return null;
    return portalsById[selectedPortalId] || null;
  }, [selectedPortalId, portalsById]);

  const hasOpenModal = useMemo(() => Object.values(openModal).some(Boolean), [openModal]);

  useEffect(() => {
    portalsRef.current = portalsById;
  }, [portalsById]);

  const addLog = useCallback((entry) => {
    setLogs((prev) => [entry, ...prev].slice(0, 120));
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
      const rows = (data?.logs || []).map((log) => ({
        id: log.id || `${log.type}-${log.timestamp}`,
        time: new Date(log.timestamp || Date.now()).toTimeString().slice(0, 5),
        type: log.type || 'LOG',
        text: log.message || ''
      }));
      setLogs(rows.slice(0, 120));
    } catch (_err) {
      setLogs([]);
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

      if (type === 'PLAYER_STATE') {
        setPlayer((prev) => {
          if (!prev) return prev;
          return {
            ...prev,
            latitude: Number(data.latitude ?? prev.latitude),
            longitude: Number(data.longitude ?? prev.longitude)
          };
        });
        return;
      }

      if (type === 'NEARBY_PLAYERS') {
        setNearbyPlayers(
          (data.players || []).map((item) => ({
            id: item.id,
            latitude: Number(item.latitude),
            longitude: Number(item.longitude)
          }))
        );
        return;
      }

      if (type === 'MAP_UPDATE' || type === 'MAP_TICK') {
        const full = Boolean(data.full);
        const mapped = (data.portals || []).map((item) =>
          mapPortalEntity({
            id: item.id,
            latitude: item.latitude,
            longitude: item.longitude,
            faction: item.faction,
            level: item.level,
            energy: item.energy
          })
        );

        if (full) {
          const next = Object.fromEntries(mapped.map((portal) => [portal.id, portal]));
          setPortalsById(next);
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
        fetchInventory().catch(() => {});
        return;
      }

      if (type === 'ERROR') {
        addLog(formatWsLog('ERROR', message));
        return;
      }

      if (type && type !== 'PONG' && type !== 'CONNECTED') {
        addLog(formatWsLog(type, message));
      }
    },
    [addLog, fetchInventory]
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
        addLog({ id: `ws-err-${Date.now()}`, time: new Date().toTimeString().slice(0, 5), type: 'ERROR', text: 'ws error' });
      }
    });
    wsRef.current = client;
    client.connect();
  }, [addLog, authTokens.accessToken, handleWsMessage]);

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
      disconnectWs();
    };
  }, [disconnectWs]);

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
    fetchInventory().catch(() => {});
    fetchLeaderboard().catch(() => {});
    fetchLogs().catch(() => {});
  }, [authReady, connectWs, fetchInventory, fetchLeaderboard, fetchLogs, fetchMapEntities, fetchPlayer, showToast]);

  useEffect(() => {
    if (!authReady) {
      disconnectWs();
    }
  }, [authReady, disconnectWs]);

  const open = (key) => setOpenModal((prev) => ({ ...prev, [key]: true }));
  const close = (key) => setOpenModal((prev) => ({ ...prev, [key]: false }));
  const closeAll = () => setOpenModal(initialModals);

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

  const handleMoveTarget = useCallback(
    ({ latitude, longitude }) => {
      setPlayer((prev) => {
        if (!prev) return prev;
        return { ...prev, latitude, longitude };
      });
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

  const sendResonatorDeploy = async (portalId, slot, level) => {
    const client = wsRef.current;
    if (!client || !isWsOpen(client)) {
      showToast('WS 未连接');
      return;
    }
    try {
      await client.request(
        'PLAYER_DEPLOY_RESONATOR',
        { portalId, slot, level, expectedVersion: 0 },
        1200
      );
      fetchInventory().catch(() => {});
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
      if (raw === 'SHIELD') return 'PORTAL_SHIELD';
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
      fetchInventory().catch(() => {});
    } catch (err) {
      showToast(err?.data?.message || err?.message || 'Mod 部署失败');
    }
  };

  const handleUpdateResonators = async (portalId, nextSlots) => {
    const current = portalsById[portalId];
    if (!current) return;
    const prevSlots = current.resonators || [];
    const idx = nextSlots.findIndex((slot, i) => {
      const prev = prevSlots[i];
      return (slot?.level || 0) !== (prev?.level || 0);
    });
    if (idx < 0) return;
    const next = nextSlots[idx];
    if (!next?.level) return;
    sendResonatorDeploy(portalId, idx + 1, Number(next.level || 1));
  };

  const handleUpdateMods = async (portalId, nextSlots) => {
    const current = portalsById[portalId];
    if (!current) return;
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
      await http.post('/api/v1/inventory/use', {
        itemType: itemId,
        amount
      });
      await fetchInventory();
      return true;
    } catch (err) {
      showToast(err?.errmsg || '道具使用失败');
      return false;
    }
  };

  const previewAttackTargets = useCallback(
    (item) => {
      const weapon = weaponForItem(item);
      if (!weapon || !player) return [];
      return attackCandidates({
        player,
        portals,
        weaponType: weapon.weaponType,
        weaponLevel: weapon.weaponLevel,
        playerFaction: player.faction
      });
    },
    [player, portals]
  );

  const attackWithFallback = async ({ weaponType, weaponLevel, portalIds, chargeBonus }) => {
    const client = wsRef.current;
    if (!client || !isWsOpen(client)) {
      throw new Error('WS 未连接');
    }

    const runSingleFallback = async () => {
      for (const portalId of portalIds) {
        client.send('PLAYER_ATTACK', {
          portalId,
          weaponType,
          weaponLevel,
          chargeBonus
        });
      }
      return 'single-fallback';
    };

    if (runtime.attackProtocol === 'single') {
      return runSingleFallback();
    }

    const batchPayload = {
      portalIds,
      weaponType,
      weaponLevel,
      chargeBonus
    };

    if (runtime.attackProtocol === 'batch') {
      await client.request('PLAYER_ATTACK', batchPayload, 900);
      return 'batch';
    }

    try {
      const response = await client.request('PLAYER_ATTACK', batchPayload, 900);
      if (response?.type === 'ERROR') {
        return runSingleFallback();
      }
      return 'batch';
    } catch (_err) {
      return runSingleFallback();
    }
  };

  const handleAttack = async ({ item, charge }) => {
    const weapon = weaponForItem(item);
    if (!weapon) {
      showToast('仅支持 XMP/US');
      return { ok: false };
    }

    const candidates = attackCandidates({
      player,
      portals,
      weaponType: weapon.weaponType,
      weaponLevel: weapon.weaponLevel,
      playerFaction: player?.faction
    });

    if (!candidates.length) {
      showToast('范围内没有可攻击 Portal');
      return { ok: false, count: 0 };
    }

    const itemType = attackItemType(weapon.weaponType, weapon.weaponLevel);
    const used = await useItem(itemType, 1);
    if (!used) return { ok: false };

    try {
      const mode = await attackWithFallback({
        weaponType: weapon.weaponType,
        weaponLevel: weapon.weaponLevel,
        portalIds: candidates.map((it) => it.portalId),
        chargeBonus: Math.max(0, Math.min(0.2, Number(charge || 0)))
      });
      showToast(`已攻击 ${candidates.length} 个目标`);
      addLog({
        id: `atk-${Date.now()}`,
        time: new Date().toTimeString().slice(0, 5),
        type: 'ATTACK',
        text: `${weapon.weaponType} L${weapon.weaponLevel} -> ${candidates.length} portals (${mode})`
      });
      return { ok: true, count: candidates.length, mode };
    } catch (err) {
      showToast(err?.message || '攻击失败');
      return { ok: false };
    }
  };

  const handlePortalClick = useCallback(
    (portal) => {
      if (linkMode.active && linkMode.fromPortalId) {
        if (portal.id === linkMode.fromPortalId) return;
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
    [linkMode, showToast]
  );

  const handleChargePortal = () => {
    const portalId = selectedPortal?.id;
    if (!portalId) return;
    const client = wsRef.current;
    if (!client || !isWsOpen(client)) {
      showToast('WS 未连接');
      return;
    }
    client
      .request('PLAYER_CHARGE_PORTAL', { portalId, amount: 500 }, 1000)
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

  return (
    <div className="screen">
      <TopBar onOpen={open} player={player} ap={playerAp} />

      <main className="map-shell">
        <MapShell
          portals={portals}
          links={links}
          fields={fields}
          player={player}
          nearbyPlayers={nearbyPlayers}
          onOpenPortal={handlePortalClick}
          onTargetUpdate={handleMoveTarget}
          onViewUpdate={handleViewUpdate}
          linkMode={linkMode}
        />
        <LogDock logs={logs} compact={hasOpenModal} />
      </main>

      <PortalModal
        open={openModal.portal}
        onClose={() => close('portal')}
        portal={selectedPortal}
        onDeploy={() => {
          close('portal');
          open('portalDeploy');
        }}
        onModDeploy={() => {
          close('portal');
          open('portalModDeploy');
        }}
        onLink={() => {
          if (!selectedPortal?.id) return;
          setLinkMode({ active: true, fromPortalId: selectedPortal.id });
          close('portal');
          showToast('Link 模式：请点击目标 Portal');
        }}
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
        playerFaction={player?.faction}
      />

      <PlayerModal open={openModal.player} onClose={() => close('player')} player={player} ap={playerAp} />

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
        onAttack={handleAttack}
        previewTargets={previewAttackTargets}
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

      {Object.values(openModal).some(Boolean) && (
        <div className="modal-backdrop" onClick={closeAll} />
      )}
    </div>
  );
}
