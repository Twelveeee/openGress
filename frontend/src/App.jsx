import React, { useMemo, useState } from 'react';
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
import { initialInventory } from './mock/inventory.js';
import gameData from './mock/game-data.json';

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

export default function App() {
  const [openModal, setOpenModal] = useState(initialModals);
  const [selectedPortalId, setSelectedPortalId] = useState(null);
  const [inventory, setInventory] = useState(initialInventory);
  const playerName = 'Twelveeee';

  const mockPortalOverrides = {
    'cf7fe928279a4fcda1270c63854c9873.16': {
      faction: 'RESISTANCE',
      owner: 'Twelveeee',
      resonators: [
        { level: 6, owner: 'Twelveeee', faction: 'RESISTANCE', xm: 100 },
        { level: 5, owner: 'Twelveeee', faction: 'RESISTANCE', xm: 92 },
        { level: 4, owner: 'Juno', faction: 'RESISTANCE', xm: 85 },
        null,
        { level: 4, owner: 'Juno', faction: 'RESISTANCE', xm: 80 },
        null,
        null,
        null
      ],
      mods: [
        { type: 'Portal Shield', subtype: 'SHIELD', rarity: 'R', owner: 'Juno' },
        { type: 'Link Amp', subtype: 'LINK', rarity: 'R', owner: 'Twelveeee' },
        null,
        null
      ]
    },
    '16f98ee4f2a64bf6ab842f6eeadb9388.16': {
      faction: 'ENLIGHTENED',
      owner: 'GreenFox',
      resonators: [
        { level: 7, owner: 'GreenFox', faction: 'ENLIGHTENED', xm: 100 },
        { level: 6, owner: 'GreenFox', faction: 'ENLIGHTENED', xm: 90 },
        { level: 6, owner: 'Nova', faction: 'ENLIGHTENED', xm: 88 },
        { level: 5, owner: 'Nova', faction: 'ENLIGHTENED', xm: 76 },
        { level: 4, owner: 'Nova', faction: 'ENLIGHTENED', xm: 72 },
        { level: 4, owner: 'GreenFox', faction: 'ENLIGHTENED', xm: 80 },
        null,
        null
      ],
      mods: [
        { type: 'Aegis Shield', subtype: 'SHIELD', rarity: 'AXA', owner: 'GreenFox' },
        { type: 'Heat Sink', subtype: 'HEATSINK', rarity: 'VR', owner: 'Nova' },
        { type: 'Multi-Hack', subtype: 'MULTI', rarity: 'R', owner: 'Nova' },
        null
      ]
    }
  };

  const initPortalState = (portal) => ({
    ...portal,
    owner: null,
    resonators: Array.from({ length: 8 }, () => null),
    mods: Array.from({ length: 4 }, () => null),
    ...(mockPortalOverrides[portal.id] || {})
  });

  const [portalState, setPortalState] = useState(() => {
    const portals = gameData.portals || [];
    return portals.reduce((acc, portal) => {
      acc[portal.id] = initPortalState(portal);
      return acc;
    }, {});
  });

  const computePortalLevel = (resonators) => {
    const total = (resonators || []).reduce((sum, slot) => sum + (slot?.level || 0), 0);
    const avg = total / 8;
    return Math.max(1, Math.floor(avg));
  };

  const selectedPortal = selectedPortalId
    ? {
        ...portalState[selectedPortalId],
        level: computePortalLevel(portalState[selectedPortalId]?.resonators)
      }
    : null;

  const portalsForMap = useMemo(() => {
    return Object.values(portalState).map((portal) => ({
      ...portal,
      level: computePortalLevel(portal.resonators)
    }));
  }, [portalState]);

  const open = (key) => setOpenModal((prev) => ({ ...prev, [key]: true }));
  const close = (key) => setOpenModal((prev) => ({ ...prev, [key]: false }));
  const closeAll = () => setOpenModal(initialModals);
  const updateItemCount = (id, delta) => {
    setInventory((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, count: Math.max(0, item.count + delta) } : item
      )
    );
  };
  const updatePortalResonators = (portalId, nextSlots) => {
    setPortalState((prev) => {
      const current = prev[portalId];
      if (!current) return prev;
      const firstRes = nextSlots.find((slot) => slot?.owner);
      const owner = firstRes?.owner || null;
      const faction = firstRes?.faction || 'NEUTRAL';
      return {
        ...prev,
        [portalId]: {
          ...current,
          owner,
          faction,
          resonators: nextSlots
        }
      };
    });
  };
  const updatePortalMods = (portalId, nextMods) => {
    setPortalState((prev) => {
      const current = prev[portalId];
      if (!current) return prev;
      return {
        ...prev,
        [portalId]: {
          ...current,
          mods: nextMods
        }
      };
    });
  };
  const recycleItems = (ids) => {
    if (!ids.length) return;
    setInventory((prev) =>
      prev.map((item) => (ids.includes(item.id) ? { ...item, count: 0 } : item))
    );
  };

  return (
    <div className="screen">
      <TopBar onOpen={open} />
      <main className="map-shell">
        <MapShell
          portals={portalsForMap}
          links={gameData.links || []}
          fields={gameData.fields || []}
          onOpenPortal={(portal) => {
            setSelectedPortalId(portal.id);
            open('portal');
          }}
        />
        <LogDock />
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
      />
      <PortalDeployModal
        open={openModal.portalDeploy}
        onClose={() => {
          close('portalDeploy');
          open('portal');
        }}
        portal={selectedPortal}
        slots={selectedPortal?.resonators}
        onUpdateSlots={updatePortalResonators}
        items={inventory}
        playerName={playerName}
      />
      <PortalModDeployModal
        open={openModal.portalModDeploy}
        onClose={() => {
          close('portalModDeploy');
          open('portal');
        }}
        portal={selectedPortal}
        modSlots={selectedPortal?.mods}
        onUpdateSlots={updatePortalMods}
        items={inventory}
        playerName={playerName}
      />
      <PlayerModal open={openModal.player} onClose={() => close('player')} />
      <InventoryModal
        open={openModal.inventory}
        onClose={() => close('inventory')}
        items={inventory}
        onRecycle={recycleItems}
      />
      <AttackModal
        open={openModal.attack}
        onClose={() => close('attack')}
        items={inventory}
        onConsume={updateItemCount}
      />
      <LeaderboardModal open={openModal.leaderboard} onClose={() => close('leaderboard')} />
      <SettingsModal open={openModal.settings} onClose={() => close('settings')} />
      <LayersModal open={openModal.layers} onClose={() => close('layers')} />

      {Object.values(openModal).some(Boolean) && (
        <div className="modal-backdrop" onClick={closeAll} />
      )}
    </div>
  );
}
