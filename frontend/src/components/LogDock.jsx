import React, { useMemo, useState } from 'react';

const classForType = (type) => {
  const key = String(type || '').toUpperCase();
  if (key.includes('RES')) return 'res';
  if (key.includes('ENL')) return 'enl';
  if (key.includes('ERROR')) return 'red';
  if (key.includes('ATTACK')) return 'red';
  if (key.includes('LINK')) return 'res';
  return '';
};

export default function LogDock({ logs = [], compact = false }) {
  const [expanded, setExpanded] = useState(true);
  const [tab, setTab] = useState('all');

  const filtered = useMemo(() => {
    if (tab === 'all') return logs;
    if (tab === 'alerts') {
      return logs.filter((log) => String(log.type || '').toUpperCase().includes('ERROR'));
    }
    return logs.filter((log) => String(log.text || '').toLowerCase().includes(tab));
  }, [logs, tab]);

  return (
    <section className={`log-dock ${expanded ? 'expanded' : 'collapsed'} ${compact ? 'compact' : ''}`}>
      <header>
        <h3>Global Log</h3>
        <button className="ghost" onClick={() => setExpanded((prev) => !prev)}>
          {expanded ? '收起' : '展开'}
        </button>
      </header>
      <div className="log-tabs">
        <button className={tab === 'all' ? 'active' : ''} onClick={() => setTab('all')}>All</button>
        <button className={tab === 'faction' ? 'active' : ''} onClick={() => setTab('faction')}>Faction</button>
        <button className={tab === 'alerts' ? 'active' : ''} onClick={() => setTab('alerts')}>Alerts</button>
      </div>
      {expanded ? (
        <div className="log-body">
          {filtered.length ? (
            filtered.map((log) => (
              <div key={log.id} className="log-row">
                <span>{log.time || '--:--'}</span>
                <span className={classForType(log.type)}>{log.type || 'LOG'}</span>
                <span>{log.text || ''}</span>
              </div>
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
