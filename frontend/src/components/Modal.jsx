import React, { useCallback, useRef, useState } from 'react';

export default function Modal({ open, onClose, className = '', placement = 'center', children }) {
  const cardRef = useRef(null);
  const dragState = useRef({ dragging: false, startX: 0, startY: 0, originX: 0, originY: 0 });
  const [position, setPosition] = useState({ x: 0, y: 0 });

  const onPointerMove = useCallback((event) => {
    if (!dragState.current.dragging) return;
    event.preventDefault();
    const dx = event.clientX - dragState.current.startX;
    const dy = event.clientY - dragState.current.startY;
    setPosition({ x: dragState.current.originX + dx, y: dragState.current.originY + dy });
  }, []);

  const onPointerUp = useCallback(() => {
    dragState.current.dragging = false;
    window.removeEventListener('pointermove', onPointerMove);
    window.removeEventListener('pointerup', onPointerUp);
  }, [onPointerMove]);

  const onPointerDown = useCallback(
    (event) => {
      if (placement === 'bottom') return;
      const handle = event.target.closest('.modal-header');
      if (!handle) return;
      event.preventDefault();
      event.stopPropagation();

      dragState.current.dragging = true;
      dragState.current.startX = event.clientX;
      dragState.current.startY = event.clientY;
      dragState.current.originX = position.x;
      dragState.current.originY = position.y;

      window.addEventListener('pointermove', onPointerMove, { passive: false });
      window.addEventListener('pointerup', onPointerUp);
    },
    [onPointerMove, onPointerUp, placement, position.x, position.y]
  );

  if (!open) return null;

  return (
    <section
      className="modal is-open"
      aria-hidden={open ? 'false' : 'true'}
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose?.();
        }
      }}
    >
      <div
        className={`modal-card placement-${placement} ${className}`}
        ref={cardRef}
        style={placement === 'bottom' ? undefined : { transform: `translate(${position.x}px, ${position.y}px)` }}
        onPointerDown={onPointerDown}
        onClick={(event) => event.stopPropagation()}
      >
        {children}
        <button className="modal-close" onClick={onClose} aria-label="Close">
          ×
        </button>
      </div>
    </section>
  );
}
