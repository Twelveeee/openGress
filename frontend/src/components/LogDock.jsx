import React, { useMemo, useState } from 'react';
import { buildLogTokens, factionClass } from '../services/logRichText.js';

const classForType = (type, faction = '') => {
  const key = String(type || '').toUpperCase();
  if (key.includes('RES')) return 'res';
  if (key.includes('ENL')) return 'enl';
  if (key.includes('ERROR')) return 'red';
  if (key.includes('ATTACK')) return 'red';
  if (key.includes('LINK') || key.includes('FIELD') || key.includes('CAPTURE')) {
    return factionClass(faction);
  }
  return '';
};

const dateKeyForTs = (timestamp) => {
  const ts = Number(timestamp);
  const date = new Date(Number.isFinite(ts) ? ts : Date.now());
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
};

export default function LogDock({
  logs = [],
  compact = false,
  playerNameById = {},
  portalNameById = {},
  portalIdByName = {},
  onPlayerClick,
  onPortalClick
}) {
  const [expanded, setExpanded] = useState(true);
  const [tab, setTab] = useState('all');

  const rows = useMemo(() => {
    let result = logs;
    if (tab === 'alerts') {
      result = logs.filter((log) => String(log.type || '').toUpperCase().includes('ERROR'));
    } else if (tab !== 'all') {
      result = logs.filter((log) => String(log.text || '').toLowerCase().includes(tab));
    }
    const sorted = [...result].sort((a, b) => Number(a.timestamp || 0) - Number(b.timestamp || 0));
    const merged = [];
    let prevDateKey = '';
    sorted.forEach((log) => {
      const currentDateKey = dateKeyForTs(log.timestamp);
      if (prevDateKey && currentDateKey !== prevDateKey) {
        merged.push({
          kind: 'divider',
          id: `divider-${currentDateKey}-${log.id}`,
          dateLabel: currentDateKey
        });
      }
      merged.push({ kind: 'log', id: log.id, log });
      prevDateKey = currentDateKey;
    });
    return merged;
  }, [logs, tab]);

  return (
    <section className={`log-dock ${expanded ? 'expanded' : 'collapsed'} ${compact ? 'compact' : ''}`}>
      <header>
        <h3>Global Log</h3>
        <button
          className={`log-toggle ${expanded ? 'expanded' : ''}`}
          aria-label={expanded ? '收起日志' : '展开日志'}
          onClick={() => setExpanded((prev) => !prev)}
        >
          <span className="log-toggle-icon">⌄</span>
        </button>
      </header>
      <div className="log-tabs">
        <button className={tab === 'all' ? 'active' : ''} onClick={() => setTab('all')}>All</button>
        <button className={tab === 'faction' ? 'active' : ''} onClick={() => setTab('faction')}>Faction</button>
        <button className={tab === 'alerts' ? 'active' : ''} onClick={() => setTab('alerts')}>Alerts</button>
      </div>
      {expanded ? (
        <div className="log-body">
          {rows.length ? (
            rows.map((row) => (
              row.kind === 'divider' ? (
                <div key={row.id} className="log-divider" role="separator" aria-label={`date ${row.dateLabel}`}>
                  <span className="log-divider-line" />
                  <span className="log-divider-label">{row.dateLabel}</span>
                  <span className="log-divider-line" />
                </div>
              ) : (
                <div key={row.id} className="log-row">
                  <span>{row.log.time || '--:--'}</span>
                  <span className={classForType(row.log.type, row.log.faction)}>{row.log.type || 'LOG'}</span>
                  <span>
                    {buildLogTokens(row.log, { playerNameById, portalNameById, portalIdByName }).map((part, idx) => {
                      if (part.clickable && part.entityType && part.entityId) {
                        const onClick = () => {
                          if (part.entityType === 'player') {
                            onPlayerClick?.(part.entityId);
                          } else if (part.entityType === 'portal') {
                            onPortalClick?.(part.entityId);
                          }
                        };
                        return (
                          <button
                            key={`${row.log.id}-token-${idx}`}
                            type="button"
                            className={`log-token-btn ${part.className || ''}`.trim()}
                            onClick={onClick}
                          >
                            {part.text}
                          </button>
                        );
                      }
                      return (
                        <span key={`${row.log.id}-token-${idx}`} className={part.className || ''}>
                          {part.text}
                        </span>
                      );
                    })}
                  </span>
                </div>
              )
            ))
          ) : (
            <div className="log-row">
              <span>--:--</span>
              <span>INFO</span>
              <span>暂无日志</span>
            </div>
          )}
        </div>
      ) : null}
    </section>
  );
}
