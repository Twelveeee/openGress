import React from 'react';

export default function NotifyLayer({ lootNotice, onCloseLoot, apNotices }) {
  const lootItems = Array.isArray(lootNotice?.items) ? lootNotice.items : [];
  const apNoticeList = Array.isArray(apNotices) ? apNotices : [];
  const leftColumn = [];
  const rightColumn = [];
  lootItems.forEach((item, idx) => {
    if (idx % 2 === 0) {
      leftColumn.push(item);
    } else {
      rightColumn.push(item);
    }
  });

  const handleLootKeyDown = (event) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onCloseLoot?.();
    }
  };

  const renderLootColumn = (items) => (
    <div className="loot-toast-col">
      {items.map((item) => (
        <div key={`${item.itemType}-${item.level || 'na'}`} className="loot-item-row">
          <span className="loot-item-icon" aria-hidden="true">{item.icon || '◇'}</span>
          <span className={`loot-item-level ${item.level ? '' : 'empty'}`}>
            {item.level ? `L${item.level}` : 'L'}
          </span>
          <span className="loot-item-count">{`x${item.count || 0}`}</span>
          <span className="loot-item-name">{item.name || item.itemType || 'Unknown'}</span>
        </div>
      ))}
    </div>
  );

  return (
    <div className="notify-layer" aria-live="polite" aria-atomic="true">
      {lootNotice ? (
        <div
          key={lootNotice.id}
          className="loot-toast"
          role="button"
          tabIndex={0}
          onClick={() => onCloseLoot?.()}
          onKeyDown={handleLootKeyDown}
          title="点击关闭"
        >
          <span className="loot-toast-header">{lootNotice.portalTitle || lootNotice.portalId || 'Portal'}</span>
          <span className="loot-toast-close" aria-hidden="true">×</span>
          <div className="loot-toast-grid">
            {renderLootColumn(leftColumn)}
            <span className="loot-toast-divider" aria-hidden="true" />
            {renderLootColumn(rightColumn)}
          </div>
        </div>
      ) : null}

      {apNoticeList.length ? (
        <div className="ap-toast-stack">
          {apNoticeList.map((notice) => (
            <div
              key={notice.id}
              className="ap-toast"
              style={{ '--ap-delay-ms': `${Number(notice.delayMs || 0)}ms` }}
            >
              {notice.text || ''}
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}
