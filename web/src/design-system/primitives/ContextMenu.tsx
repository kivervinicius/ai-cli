import React, { useEffect, useLayoutEffect, useRef } from 'react';
import { createPortal } from 'react-dom';

export type ContextMenuPoint = { x: number; y: number };

export type ContextMenuItem =
  | { type: 'separator'; id: string }
  | {
      type: 'item';
      id: string;
      label: string;
      icon?: React.ReactNode;
      danger?: boolean;
      disabled?: boolean;
      onSelect: () => void;
    };

export function contextMenuFromEvent(event: React.MouseEvent): ContextMenuPoint {
  event.preventDefault();
  event.stopPropagation();
  return { x: event.clientX, y: event.clientY };
}

export function clampMenuPosition(
  x: number,
  y: number,
  width: number,
  height: number,
  viewportWidth: number,
  viewportHeight: number,
  pad = 8
): ContextMenuPoint {
  return {
    x: Math.min(Math.max(pad, Math.round(x)), Math.max(pad, viewportWidth - width - pad)),
    y: Math.min(Math.max(pad, Math.round(y)), Math.max(pad, viewportHeight - height - pad)),
  };
}

export const ContextMenu: React.FC<{
  open: ContextMenuPoint | null;
  onClose: () => void;
  label: string;
  items: ContextMenuItem[];
}> = ({ open, onClose, label, items }) => {
  const menuRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const menu = menuRef.current;
    if (!open || !menu) return;
    const rect = menu.getBoundingClientRect();
    const next = clampMenuPosition(open.x, open.y, rect.width, rect.height, window.innerWidth, window.innerHeight);
    menu.style.left = `${next.x}px`;
    menu.style.top = `${next.y}px`;
  }, [open, items]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      if (menuRef.current?.contains(event.target as Node)) return;
      onClose();
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('pointerdown', onPointerDown, true);
    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('pointerdown', onPointerDown, true);
      window.removeEventListener('keydown', onKey);
    };
  }, [open, onClose]);

  if (!open) return null;

  return createPortal(
    <div
      ref={menuRef}
      className="nx-context-menu"
      role="menu"
      aria-label={label}
      style={{ left: open.x, top: open.y }}
      onPointerDown={(event) => event.stopPropagation()}
      onContextMenu={(event) => event.preventDefault()}
    >
      {items.map((item) =>
        item.type === 'separator' ? (
          <div key={item.id} className="nx-context-menu__sep" role="separator" />
        ) : (
          <button
            key={item.id}
            type="button"
            role="menuitem"
            className="nx-context-menu__item"
            data-danger={item.danger ? 'true' : undefined}
            disabled={item.disabled}
            onClick={() => {
              if (item.disabled) return;
              item.onSelect();
              onClose();
            }}
          >
            {item.icon}
            <span>{item.label}</span>
          </button>
        )
      )}
    </div>,
    document.body
  );
};
