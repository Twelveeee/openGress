import React, { useState } from 'react';

export default function LogDock() {
  const [expanded, setExpanded] = useState(true);
  const [tab, setTab] = useState('all');

  return (
    <section className={`log-dock ${expanded ? 'expanded' : 'collapsed'}`}>
      <header>
        <h3>Global Log</h3>
        <button className="ghost" onClick={() => setExpanded((prev) => !prev)}>
          {expanded ? '收起' : '展开'}
        </button>
      </header>
      <div className="log-tabs">
        <button className={tab === 'all' ? 'active' : ''} onClick={() => setTab('all')}>
          All
        </button>
        <button className={tab === 'faction' ? 'active' : ''} onClick={() => setTab('faction')}>
          Faction
        </button>
        <button className={tab === 'alerts' ? 'active' : ''} onClick={() => setTab('alerts')}>
          Alerts
        </button>
      </div>
      {expanded && (
        <div className="log-body">
          <div className="log-row">
            <span>22:44</span>
            <span className="res">Deploy</span>
            <span>AgentNova → Portal Zenith</span>
          </div>
          <div className="log-row">
            <span>22:41</span>
            <span className="enl">Link</span>
            <span>Cipher → Aurora Gate</span>
          </div>
          <div className="log-row">
            <span>22:39</span>
            <span className="red">Neutralize</span>
            <span>Nadir → Redline Hub</span>
          </div>
        </div>
      )}
    </section>
  );
}
