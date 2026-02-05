import React, { useEffect, useRef } from 'react';
import { getInventoryIcon } from '../utils/itemIcons.js';

function levelClass(level) {
  if (level <= 3) return 'lvl-low';
  if (level <= 6) return 'lvl-mid';
  return 'lvl-high';
}

export default function ItemUseBar({
  helpText,
  actionLabel,
  items = [],
  selectedId,
  onSelect,
  onAction,
  actionDisabled = false,
  actionDisabledText,
  actionClassName = '',
  footerText,
  notice,
  noticeKey,
  itemMode,
  onActionMouseDown,
  onActionMouseUp,
  onActionMouseLeave,
  onActionTouchStart,
  onActionTouchEnd
}) {
  const listRef = useRef(null);
  const dragRef = useRef({ down: false, startX: 0, scrollLeft: 0 });

  const handleMouseDown = (event) => {
    const el = listRef.current;
    if (!el) return;
    dragRef.current.down = true;
    dragRef.current.startX = event.clientX;
    dragRef.current.scrollLeft = el.scrollLeft;
  };

  const handleMouseMove = (event) => {
    if (!dragRef.current.down) return;
    const el = listRef.current;
    if (!el) return;
    const dx = event.clientX - dragRef.current.startX;
    el.scrollLeft = dragRef.current.scrollLeft - dx;
  };

  const stopDrag = () => {
    dragRef.current.down = false;
  };

  useEffect(() => {
    const el = listRef.current;
    if (!el) return undefined;
    const handleWheel = (event) => {
      if (Math.abs(event.deltaY) > Math.abs(event.deltaX)) {
        event.preventDefault();
        el.scrollLeft += event.deltaY;
      }
    };
    el.addEventListener('wheel', handleWheel, { passive: false });
    return () => el.removeEventListener('wheel', handleWheel);
  }, []);

  return (
    <section className="item-use">
      <div className="item-use-top">
        <button className="item-use-info" aria-label="Help" title={helpText || ''}>
          ?
          {helpText ? <span className="item-use-tooltip">{helpText}</span> : null}
        </button>
        <div className="item-use-action-wrap">
          {notice ? (
            <div key={noticeKey} className="item-use-notice">
              {notice}
            </div>
          ) : null}
          <button
            className={`item-use-action ${actionClassName} ${actionDisabled ? 'disabled' : ''}`}
            onClick={onAction}
            disabled={actionDisabled}
            onMouseDown={onActionMouseDown}
            onMouseUp={onActionMouseUp}
            onMouseLeave={onActionMouseLeave}
            onTouchStart={onActionTouchStart}
            onTouchEnd={onActionTouchEnd}
          >
            {actionDisabled && actionDisabledText ? actionDisabledText : actionLabel}
          </button>
        </div>
        <div className="item-use-spacer" />
      </div>

      <div
        className="item-use-list"
        ref={listRef}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={stopDrag}
        onMouseLeave={stopDrag}
      >
        {items.map((item) => {
          const icon = getInventoryIcon(item);
          return (
            <button
              key={item.id}
              className={`item-use-card ${item.id === selectedId ? 'active' : ''}`}
              onClick={() => onSelect?.(item.id)}
            >
              <div className={`item-use-sprite ${icon ? 'has-icon' : ''}`}>
                {icon ? <img className="item-use-icon" src={icon} alt={item.name || 'item'} /> : null}
              </div>
              <div className="item-use-meta">
                {itemMode === 'rarity' || item.type === 'mod' ? (
                  <span className={`rarity ${item.rarity || 'C'}`}>///</span>
                ) : (
                  <span className={`item-use-level ${levelClass(item.level || 0)}`}>
                    L{item.level || 0}
                  </span>
                )}
                <span className="item-use-count">x{item.count ?? 0}</span>
              </div>
            </button>
          );
        })}
      </div>
      {footerText ? <div className="item-use-footer">{footerText}</div> : null}
    </section>
  );
}
