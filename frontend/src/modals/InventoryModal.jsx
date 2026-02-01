import React, { useMemo, useState } from 'react';
import Modal from '../components/Modal.jsx';

const tabs = [
  { key: 'weapon', label: 'Weapons' },
  { key: 'resonator', label: 'Resonators' },
  { key: 'mod', label: 'Mods' },
  { key: 'key', label: 'Keys' },
  { key: 'cube', label: 'Cubes' }
];

function levelClass(level) {
  if (level <= 3) return 'lvl-low';
  if (level <= 6) return 'lvl-mid';
  return 'lvl-high';
}

function factionClass(faction) {
  switch (faction) {
    case 'RESISTANCE':
      return 'faction-res';
    case 'ENLIGHTENED':
      return 'faction-enl';
    default:
      return 'faction-neutral';
  }
}

export default function InventoryModal({ open, onClose, items = [], onRecycle }) {
  const [tab, setTab] = useState('weapon');
  const [selectedId, setSelectedId] = useState(null);
  const [manageMode, setManageMode] = useState(false);
  const [selected, setSelected] = useState([]);
  const [toast, setToast] = useState('');
  const [sortKey, setSortKey] = useState('distance');
  const [sortOpen, setSortOpen] = useState(false);
  const [favorites, setFavorites] = useState([]);

  const filtered = useMemo(
    () => items.filter((item) => item.type === tab && item.count > 0),
    [items, tab]
  );
  const focused = filtered.find((item) => item.id === selectedId) || filtered[0];

  const toggleSelect = (id) => {
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  };

  const recycle = () => {
    if (!selected.length) return;
    onRecycle?.(selected);
    setSelected([]);
    setToast('回收成功');
    setTimeout(() => setToast(''), 1200);
  };

  const toggleFavorite = (id) => {
    setFavorites((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  };

  const sorted = useMemo(() => {
    if (tab !== 'key') return filtered;
    const currentFaction = 'RESISTANCE';
    const compareDistance = (a, b) => (a.distance || 0) - (b.distance || 0);
    const compareName = (a, b) => (a.name || '').localeCompare(b.name || '', 'zh');
    const compareFaction = (a, b) => {
      const aOwn = a.faction === currentFaction ? 0 : 1;
      const bOwn = b.faction === currentFaction ? 0 : 1;
      if (aOwn !== bOwn) return aOwn - bOwn;
      return compareDistance(a, b);
    };

    const sorter = (a, b) => {
      if (sortKey === 'name') return compareName(a, b);
      if (sortKey === 'faction') return compareFaction(a, b);
      return compareDistance(a, b);
    };

    const fav = filtered.filter((item) => favorites.includes(item.id)).sort(sorter);
    const rest = filtered.filter((item) => !favorites.includes(item.id)).sort(sorter);
    return [...fav, ...rest];
  }, [filtered, favorites, sortKey, tab]);

  const formatDistance = (meters) => {
    if (meters == null) return '—';
    if (meters >= 1000) return `${(meters / 1000).toFixed(1)} km`;
    return `${meters.toFixed(1)} m`;
  };

  return (
    <Modal open={open} onClose={onClose} className="inventory-card">
      <header className="modal-header">
        <h3>库存管理</h3>
      </header>
      <div className="inventory-body">
        <div className="inventory-tabs">
          {tabs.map((t) => (
            <button
              key={t.key}
              className={`tab ${tab === t.key ? 'active' : ''}`}
              onClick={() => {
                setTab(t.key);
                setSelectedId(null);
                setSelected([]);
              }}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className="inventory-list">
          {(tab === 'key' ? sorted : filtered).map((item) => (
            <label
              key={item.id}
              className={`inventory-row ${focused?.id === item.id ? 'active' : ''}`}
            >
              {manageMode ? (
                <input
                  type="checkbox"
                  checked={selected.includes(item.id)}
                  onChange={() => toggleSelect(item.id)}
                />
              ) : null}
              <button className="row-content" onClick={() => setSelectedId(item.id)}>
                {item.type === 'key' ? (
                  <span className="key-layout right-metrics">
                    <span className="key-line key-line--title">
                      <span className={`item-level ${levelClass(item.level || 0)}`}>
                        L{item.level || 0}
                      </span>
                      <span className="item-name">{item.name}</span>
                    </span>
                    <span className="key-line">
                      <span className="key-address">{item.address || '—'}</span>
                    </span>
                    <span className="key-line">
                      <span className="key-distance">
                        {formatDistance(item.distance)} · x{item.count}
                      </span>
                    </span>
                    <span className="key-metrics">
                      <button
                        className={`star ${favorites.includes(item.id) ? 'fav' : ''}`}
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleFavorite(item.id);
                        }}
                        aria-label="favorite"
                      >
                        {favorites.includes(item.id) ? '★' : '☆'}
                      </button>
                      <span className={`res-bars ${factionClass(item.faction)}`}>
                        {(item.resonators || []).map((val, idx) => (
                          <span key={idx} className="res-bar">
                            <span className="res-bar-fill" style={{ height: `${val}%` }} />
                            {val >= 100 ? <span className="res-bar-cap" /> : null}
                          </span>
                        ))}
                      </span>
                    </span>
                  </span>
                ) : (
                  <>
                    <span className="item-name">{item.name}</span>
                    <span className="item-meta">
                      {item.type === 'mod' || item.subtype === 'VIRUS' ? (
                        <span className={`rarity ${item.rarity || 'C'}`} aria-label={item.rarity}>
                          ///
                        </span>
                      ) : (
                        <span className={`item-level ${levelClass(item.level || 0)}`}>
                          L{item.level || 0}
                        </span>
                      )}
                      <span className="item-count">x{item.count}</span>
                    </span>
                  </>
                )}
              </button>
            </label>
          ))}
        </div>

        <div className="inventory-count">
          <div>
            <div className="inventory-selected">{focused ? focused.name : '—'}</div>
            <div className="inventory-sub">数量: {focused?.count ?? 0}</div>
          </div>
          {manageMode ? (
            <div className="manage-actions">
              {tab === 'key' ? (
                <div className="key-sort">
                  <button
                    className="ghost sort-trigger"
                    onClick={() => setSortOpen((prev) => !prev)}
                  >
                    {sortKey} ▾
                  </button>
                  {sortOpen ? (
                    <div className="sort-menu">
                      {['distance', 'faction', 'name'].map((key) => (
                        <button
                          key={key}
                          className={`ghost ${sortKey === key ? 'active' : ''}`}
                          onClick={() => {
                            setSortKey(key);
                            setSortOpen(false);
                          }}
                        >
                          {key}
                        </button>
                      ))}
                    </div>
                  ) : null}
                </div>
              ) : null}
              <button className="ghost" onClick={() => setSelected(filtered.map((i) => i.id))}>
                全选
              </button>
              <button className="ghost" onClick={() => setSelected([])}>
                清空
              </button>
              <button className="primary" onClick={recycle}>
                回收所选
              </button>
              <button className="ghost" onClick={() => setManageMode(false)}>
                退出管理
              </button>
            </div>
          ) : (
            <div className="manage-actions">
              {tab === 'key' ? (
                <div className="key-sort">
                  <button
                    className="ghost sort-trigger"
                    onClick={() => setSortOpen((prev) => !prev)}
                  >
                    {sortKey} ▾
                  </button>
                  {sortOpen ? (
                    <div className="sort-menu">
                      {['distance', 'faction', 'name'].map((key) => (
                        <button
                          key={key}
                          className={`ghost ${sortKey === key ? 'active' : ''}`}
                          onClick={() => {
                            setSortKey(key);
                            setSortOpen(false);
                          }}
                        >
                          {key}
                        </button>
                      ))}
                    </div>
                  ) : null}
                </div>
              ) : null}
              <button className="primary" onClick={() => setManageMode(true)}>
                Manage
              </button>
            </div>
          )}
        </div>
      </div>
      {toast ? <div className="toast">{toast}</div> : null}
    </Modal>
  );
}
