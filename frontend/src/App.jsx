import React, { useState } from 'react';
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
  const [selectedPortal, setSelectedPortal] = useState(null);
  const [inventory, setInventory] = useState(initialInventory);
  const playerName = 'Twelveeee';

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
          onOpenPortal={(portal) => {
            setSelectedPortal(portal);
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
        onClose={() => close('portalDeploy')}
        portal={selectedPortal}
        items={inventory}
        playerName={playerName}
      />
      <PortalModDeployModal
        open={openModal.portalModDeploy}
        onClose={() => close('portalModDeploy')}
        portal={selectedPortal}
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
