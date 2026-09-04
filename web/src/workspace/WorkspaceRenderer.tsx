import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { AppWindow, Gauge, History, Home, Layers, LayoutGrid, Maximize2, Minimize2, Minus, Paintbrush, PanelsTopLeft, Settings, Sparkles, TerminalSquare, Workflow, X } from 'lucide-react';
import { IconButton } from '../design-system';
import { listStacks, listSurfaces, surfaceViewId, type WorkspaceStack, type WorkspaceSurface } from './model';
import { useWorkspace } from './WorkspaceProvider';
import { useWorkspacePresentation } from './WorkspacePresentationProvider';
import { useTranslation } from 'react-i18next';
import { isPtySurface } from '../app/surfaces';
import { findTileSplitters } from './arrange';
import { isWindowedPresentationMode } from './presentation';

const WINDOW_ACCENTS = ['#38bdf8', '#22c55e', '#f59e0b', '#f472b6', '#a78bfa', '#fb7185'];
const WINDOW_ICONS = ['⌘', '⚡', '◆', '●', '★', '◎'];

const PRODUCT_TAB_ICONS: Record<string, React.ComponentType<{ size?: number }>> = {
  overview: Home,
  terminals: TerminalSquare,
  work: Sparkles,
  missions: Workflow,
  agents: PanelsTopLeft,
  settings: Settings,
  resources: Gauge,
  sessions: History,
  projects: LayoutGrid,
};

const isPinnedProductTab = (surface: WorkspaceSurface) =>
  surface.closable === false || surface.type === 'overview' || surface.type === 'terminals' || surface.type === 'work';

const WindowChromeMenu: React.FC<{
  viewId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customTitle: string;
  accent: string;
  icon: string;
  onPatch: (chrome: { customTitle?: string; accent?: string; icon?: string }) => void;
}> = ({ open, onOpenChange, customTitle, accent, icon, onPatch }) => {
  const rootRef = useRef<HTMLDivElement>(null);
  const [draft, setDraft] = useState(customTitle);
  const [menuPos, setMenuPos] = useState<{ top: number; right: number } | null>(null);

  useEffect(() => {
    setDraft(customTitle);
  }, [customTitle]);

  useEffect(() => {
    if (!open) {
      setMenuPos(null);
      return;
    }
    const anchor = rootRef.current?.querySelector('button');
    if (anchor) {
      const rect = anchor.getBoundingClientRect();
      setMenuPos({
        top: rect.bottom + 6,
        right: Math.max(8, window.innerWidth - rect.right),
      });
    }
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      if ((target as Element).closest?.('.nx-desktop-window__chrome-menu')) return;
      onOpenChange(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onOpenChange(false);
    };
    window.addEventListener('pointerdown', onPointerDown, true);
    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('pointerdown', onPointerDown, true);
      window.removeEventListener('keydown', onKey);
    };
  }, [open, onOpenChange]);

  const commitTitle = () => {
    onPatch({ customTitle: draft.trim() });
  };

  const menu =
    open && menuPos
      ? createPortal(
          <div
            className="nx-desktop-window__chrome-menu"
            role="dialog"
            aria-label="Personalizar janela"
            style={{ top: menuPos.top, right: menuPos.right }}
            onPointerDown={(event) => event.stopPropagation()}
          >
            <label>
              Nome
              <input
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                onBlur={commitTitle}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    commitTitle();
                    onOpenChange(false);
                  }
                }}
                placeholder="Identidade da janela"
                autoFocus
              />
            </label>
            <div className="nx-desktop-window__chrome-row" aria-label="Cor do título">
              <button
                type="button"
                className="nx-desktop-window__swatch nx-desktop-window__swatch--clear"
                data-active={!accent ? 'true' : 'false'}
                title="Sem cor"
                onClick={() => {
                  onPatch({ accent: '' });
                  onOpenChange(false);
                }}
              />
              {WINDOW_ACCENTS.map((color) => (
                <button
                  key={color}
                  type="button"
                  className="nx-desktop-window__swatch"
                  data-active={accent === color ? 'true' : 'false'}
                  style={{ background: color }}
                  title={color}
                  onClick={() => {
                    onPatch({ accent: color });
                    onOpenChange(false);
                  }}
                />
              ))}
            </div>
            <div className="nx-desktop-window__chrome-row" aria-label="Ícone">
              <button
                type="button"
                className="nx-desktop-window__icon-pick"
                data-active={!icon ? 'true' : 'false'}
                title="Sem ícone"
                onClick={() => {
                  onPatch({ icon: '' });
                  onOpenChange(false);
                }}
              >
                —
              </button>
              {WINDOW_ICONS.map((glyph) => (
                <button
                  key={glyph}
                  type="button"
                  className="nx-desktop-window__icon-pick"
                  data-active={icon === glyph ? 'true' : 'false'}
                  onClick={() => {
                    onPatch({ icon: glyph });
                    onOpenChange(false);
                  }}
                >
                  {glyph}
                </button>
              ))}
            </div>
          </div>,
          document.body
        )
      : null;

  return (
    <div className="nx-desktop-window__chrome" ref={rootRef}>
      <IconButton
        label="Personalizar janela"
        aria-expanded={open}
        onClick={(event) => {
          event.stopPropagation();
          onOpenChange(!open);
        }}
      >
        <Paintbrush size={13} />
      </IconButton>
      {menu}
    </div>
  );
};

export type WorkspaceCreateActions = {
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
};

function useCompactViewport(): boolean {
  const [compact, setCompact] = useState(() => window.matchMedia('(max-width: 720px)').matches);
  useEffect(() => {
    const media = window.matchMedia('(max-width: 720px)');
    const update = () => setCompact(media.matches);
    media.addEventListener?.('change', update);
    return () => media.removeEventListener?.('change', update);
  }, []);
  return compact;
}

export const WorkspaceRenderer: React.FC<{
  renderSurface: (surface: WorkspaceSurface) => React.ReactNode;
  onRequestClose?: (surface: WorkspaceSurface) => void;
  /** @deprecated CTAs live in the topbar/rail; kept optional for call-site compat. */
  createActions?: WorkspaceCreateActions;
}> = ({ renderSurface, onRequestClose }) => {
  const workspace = useWorkspace();
  const presentation = useWorkspacePresentation();
  const compact = useCompactViewport();
  const surfaces = useMemo(() => listSurfaces(workspace.model.root), [workspace.model.root]);
  const floatingSurfaces = useMemo(() => surfaces.filter(isPtySurface), [surfaces]);
  const surfaceSignature = floatingSurfaces.map(surfaceViewId).join('|');
  const windowSignature = Object.keys(presentation.state.windows).sort().join('|');

  useEffect(() => {
    if (surfaceSignature !== windowSignature) presentation.sync(floatingSurfaces);
  }, [surfaceSignature, windowSignature, floatingSurfaces, presentation]);

  const stacks = listStacks(workspace.model.root);
  const activeStack = stacks.find((stack) => stack.activeId) ?? stacks[0];

  return (
    <div className={compact ? 'nx-workspace nx-workspace--compact' : 'nx-workspace'} data-tour="workspace">
      {activeStack && (
        <WorkspaceStackView
          stack={activeStack}
          renderSurface={renderSurface}
          onRequestClose={onRequestClose}
        />
      )}
    </div>
  );
};

const WorkspaceStackView: React.FC<{
  stack: WorkspaceStack;
  renderSurface: (surface: WorkspaceSurface) => React.ReactNode;
  onRequestClose?: (surface: WorkspaceSurface) => void;
}> = ({ stack, renderSurface, onRequestClose }) => {
  const { t } = useTranslation();
  const { activate, close, move } = useWorkspace();
  const [draggedSurface, setDraggedSurface] = useState<string | null>(null);
  const productTabs = useMemo(() => {
    const pinnedOrder = ['overview', 'terminals', 'work'];
    return [...stack.tabs.filter((tab) => !isPtySurface(tab))].sort((a, b) => {
      const ai = pinnedOrder.indexOf(a.type);
      const bi = pinnedOrder.indexOf(b.type);
      if (ai === -1 && bi === -1) return 0;
      if (ai === -1) return 1;
      if (bi === -1) return -1;
      return ai - bi;
    });
  }, [stack.tabs]);
  const ptyTabs = useMemo(() => stack.tabs.filter(isPtySurface), [stack.tabs]);
  const activeRaw = stack.tabs.find((tab) => tab.id === stack.activeId);
  // If a PTY was left as activeId (legacy layouts), treat the Terminais product tab as active.
  const activeProduct =
    activeRaw && !isPtySurface(activeRaw)
      ? activeRaw
      : productTabs.find((tab) => tab.type === 'terminals') || productTabs[0];
  const activeId = activeProduct?.id || stack.activeId;
  const canClose = activeProduct?.closable !== false;
  const closeSurface = (surface: WorkspaceSurface) => (onRequestClose ? onRequestClose(surface) : close(surface.id));
  const legacyTitleKeys: Record<string, string> = {
    overview: 'nav.overview',
    work: 'nav.work',
    missions: 'nav.missions',
    terminals: 'nav.terminals',
    agents: 'nav.agents',
    maestro: 'nav.maestro',
    sessions: 'nav.sessions',
    settings: 'nav.settings',
    resources: 'nav.resources',
    'legacy-runtimes': 'nav.runtimes',
    'legacy-providers': 'nav.providers',
    'legacy-events': 'nav.events',
  };
  const displayTitle = (surface: WorkspaceSurface) => {
    const key = surface.titleKey || legacyTitleKeys[surface.type];
    return key ? t(key, surface.titleParams) : surface.title;
  };

  return (
    <section
      className="nx-workspace-stack"
      data-stack-id={stack.id}
      onDragOver={(event) => {
        if (event.dataTransfer.types.includes('application/x-nexus-surface')) event.preventDefault();
      }}
      onDrop={(event) => {
        const id = event.dataTransfer.getData('application/x-nexus-surface') || draggedSurface;
        if (id) move(id, stack.id);
        setDraggedSurface(null);
      }}
    >
      <header className="nx-workspace-stack__tabs">
        <div className="nx-workspace-tabs" role="tablist" aria-label={t('shell.openSurfaces')}>
          {productTabs.map((surface) => {
            const TabIcon = PRODUCT_TAB_ICONS[surface.type] || Layers;
            const pinned = isPinnedProductTab(surface);
            return (
            <button
              draggable={!pinned}
              key={surface.id}
              type="button"
              role="tab"
              aria-selected={activeId === surface.id}
              data-active={activeId === surface.id ? 'true' : 'false'}
              data-pinned={pinned ? 'true' : undefined}
              data-kind={surface.type}
              data-attention={surface.data?.hasAttention === 'true' ? 'true' : undefined}
              data-unread={surface.data?.unreadAttention === 'true' ? 'true' : undefined}
              className="nx-workspace-tab"
              onDragStart={(event) => {
                if (pinned) {
                  event.preventDefault();
                  return;
                }
                setDraggedSurface(surface.id);
                event.dataTransfer.setData('application/x-nexus-surface', surface.id);
                event.dataTransfer.effectAllowed = 'move';
              }}
              onClick={() => activate(surface.id)}
            >
              <span className="nx-workspace-tab__icon" aria-hidden="true">
                <TabIcon size={14} />
              </span>
              {(surface.data?.hasAttention === 'true' || surface.data?.unreadAttention === 'true') && (
                <span
                  className="nx-workspace-tab__attention-dot"
                  data-kind={surface.data?.attentionKind || undefined}
                  data-unread={surface.data?.unreadAttention === 'true' ? 'true' : undefined}
                  aria-label="needs attention"
                />
              )}
              <span className="nx-workspace-tab__label">{displayTitle(surface)}</span>
              {surface.closable !== false && (
                <span
                  className="nx-workspace-tab__close"
                  role="button"
                  aria-label={t('workspace.closeNamed', { name: displayTitle(surface) })}
                  onClick={(event) => {
                    event.stopPropagation();
                    closeSurface(surface);
                  }}
                >
                  <X size={11} />
                </span>
              )}
            </button>
            );
          })}
        </div>
        {activeProduct && canClose && (
          <div className="nx-workspace-stack__actions">
            <IconButton label={t('workspace.close')} onClick={() => closeSurface(activeProduct)}>
              <X size={13} />
            </IconButton>
          </div>
        )}
      </header>
      <div className="nx-workspace-stack__body">
        {productTabs.map((surface) => (
          <div
            key={surface.id}
            role="tabpanel"
            aria-hidden={activeId !== surface.id}
            data-active={activeId === surface.id ? 'true' : 'false'}
            className="nx-workspace-panel"
          >
            {surface.type === 'terminals' ? (
              <TerminalsHost
                ptySurfaces={ptyTabs}
                renderSurface={renderSurface}
                onRequestClose={onRequestClose}
                active={activeId === surface.id}
              />
            ) : (
              renderSurface(surface)
            )}
          </div>
        ))}
      </div>
    </section>
  );
};

/** Owns all PTYs so they stay mounted when other product tabs are active. */
const TerminalsHost: React.FC<{
  ptySurfaces: WorkspaceSurface[];
  renderSurface: (surface: WorkspaceSurface) => React.ReactNode;
  onRequestClose?: (surface: WorkspaceSurface) => void;
  active: boolean;
}> = ({ ptySurfaces, renderSurface, onRequestClose, active }) => {
  const { t } = useTranslation();
  const presentation = useWorkspacePresentation();
  const focusedPtyId =
    ptySurfaces.find((s) => surfaceViewId(s) === presentation.state.activePtyViewId)?.id ||
    ptySurfaces[0]?.id ||
    '';
  const windowed = isWindowedPresentationMode(presentation.state.mode);

  useEffect(() => {
    const activePtyViewId = presentation.state.activePtyViewId;
    const setActivePty = presentation.setActivePty;
    if (ptySurfaces.length === 0) {
      if (activePtyViewId) setActivePty('');
      return;
    }
    const viewIds = ptySurfaces.map((s) => surfaceViewId(s));
    if (!activePtyViewId || !viewIds.includes(activePtyViewId)) {
      setActivePty(viewIds[0] || '');
    }
  }, [ptySurfaces, presentation.state.activePtyViewId, presentation.setActivePty]);

  const focusPty = (surface: WorkspaceSurface) => {
    const viewId = surfaceViewId(surface);
    presentation.setActivePty(viewId);
    if (windowed) {
      presentation.focus(viewId);
    }
  };

  return (
    <div className="nx-terminals-host" data-active={active ? 'true' : 'false'}>
      <header className="nx-terminals-host__chrome">
        <div className="nx-terminals-host__title">
          <TerminalSquare size={14} />
          <strong>{t('nav.terminals')}</strong>
          <small>
            {ptySurfaces.length} PTY{ptySurfaces.length === 1 ? '' : 's'}
          </small>
        </div>
        <div className="nx-presentation-toggle" role="group" aria-label="Terminal presentation">
          <button
            type="button"
            data-active={presentation.state.mode === 'TABS' ? 'true' : 'false'}
            onClick={() => presentation.setMode('TABS')}
            title="Abas internas de PTY"
          >
            <PanelsTopLeft size={11} />
            <span>Abas</span>
          </button>
          <button
            type="button"
            data-active={presentation.state.mode === 'DESKTOP' ? 'true' : 'false'}
            onClick={() => presentation.setMode('DESKTOP')}
            title="Janelas flutuantes"
          >
            <AppWindow size={11} />
            <span>Janelas</span>
          </button>
          <button
            type="button"
            data-active={presentation.state.mode === 'MOSAIC' ? 'true' : 'false'}
            onClick={() => presentation.setMode('MOSAIC')}
            title="Mosaico lado a lado"
          >
            <LayoutGrid size={11} />
            <span>Mosaico</span>
          </button>
        </div>
      </header>
      <div className="nx-terminals-host__body">
        {ptySurfaces.length === 0 ? (
          <div className="nx-terminals-empty">
            <TerminalSquare size={22} />
            <strong>Nenhum terminal aberto</strong>
            <p>Use Novo Agente, Sessão IA ou Terminal no topbar ou no rail para criar um PTY nesta aba.</p>
          </div>
        ) : windowed ? (
          <DesktopWorkspace surfaces={ptySurfaces} renderSurface={renderSurface} onRequestClose={onRequestClose} />
        ) : (
          <div className="nx-terminals-inner-tabs">
            <div className="nx-workspace-tabs" role="tablist" aria-label="PTY tabs">
              {ptySurfaces.map((surface) => (
                <button
                  key={surface.id}
                  type="button"
                  role="tab"
                  className="nx-workspace-tab"
                  data-active={focusedPtyId === surface.id ? 'true' : 'false'}
                  data-attention={surface.data?.hasAttention === 'true' ? 'true' : undefined}
                  data-unread={surface.data?.unreadAttention === 'true' ? 'true' : undefined}
                  onClick={() => focusPty(surface)}
                >
                  <span>{surface.data?.agentName || surface.title}</span>
                  {surface.closable !== false && onRequestClose && (
                    <span
                      className="nx-workspace-tab__close"
                      role="button"
                      onClick={(event) => {
                        event.stopPropagation();
                        onRequestClose(surface);
                      }}
                    >
                      <X size={11} />
                    </span>
                  )}
                </button>
              ))}
            </div>
            <div className="nx-terminals-inner-panels">
              {ptySurfaces.map((surface) => (
                <div
                  key={surface.id}
                  className="nx-workspace-panel"
                  data-active={focusedPtyId === surface.id ? 'true' : 'false'}
                  aria-hidden={focusedPtyId !== surface.id}
                >
                  {renderSurface(surface)}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

const DesktopWorkspace: React.FC<{
  surfaces: WorkspaceSurface[];
  renderSurface: (surface: WorkspaceSurface) => React.ReactNode;
  onRequestClose?: (surface: WorkspaceSurface) => void;
}> = ({ surfaces, renderSurface, onRequestClose }) => {
  const workspace = useWorkspace();
  const presentation = useWorkspacePresentation();
  const hostRef = useRef<HTMLDivElement>(null);
  const [chromeMenuViewId, setChromeMenuViewId] = useState<string | null>(null);
  const closeSurface = (surface: WorkspaceSurface) => (onRequestClose ? onRequestClose(surface) : workspace.close(surface.id));
  const terminalSurfaces = surfaces.filter(isPtySurface);
  const mosaic = presentation.state.mode === 'MOSAIC';

  const windows = terminalSurfaces
    .map((surface) => ({ surface, win: presentation.state.windows[surfaceViewId(surface)] }))
    .filter((item) => Boolean(item.win))
    .sort((a, b) => (a.win?.zIndex ?? 0) - (b.win?.zIndex ?? 0));

  const splitters = useMemo(() => {
    if (!mosaic) return [];
    const tiles = windows
      .filter(({ win }) => win && !win.minimized && !win.maximized)
      .map(({ win }) => win!);
    return findTileSplitters(tiles);
  }, [mosaic, windows]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const publish = () => {
      const rect = host.getBoundingClientRect();
      presentation.setCanvas({
        x: 8,
        y: 8,
        width: Math.max(320, Math.floor(rect.width - 16)),
        height: Math.max(240, Math.floor(rect.height - 16)),
      });
    };
    publish();
    const observer = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(() => publish()) : null;
    observer?.observe(host);
    return () => observer?.disconnect();
  }, [presentation]);

  const startMove = (viewId: string, event: React.PointerEvent) => {
    if (event.button !== 0) return;
    const win = presentation.state.windows[viewId];
    if (!win || win.maximized || win.minimized) return;
    event.preventDefault();
    event.stopPropagation();
    presentation.focus(viewId);
    const startX = event.clientX;
    const startY = event.clientY;
    const origin = { x: win.x, y: win.y, width: win.width, height: win.height };
    const onMove = (move: PointerEvent) => {
      presentation.move(viewId, origin.x + (move.clientX - startX), origin.y + (move.clientY - startY));
    };
    const onUp = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
      if (mosaic) presentation.commitMove(viewId, origin);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp, { once: true });
  };

  const startResize = (viewId: string, event: React.PointerEvent) => {
    if (mosaic || event.button !== 0) return;
    const win = presentation.state.windows[viewId];
    if (!win || win.maximized || win.minimized) return;
    event.preventDefault();
    event.stopPropagation();
    presentation.focus(viewId);
    const startX = event.clientX;
    const startY = event.clientY;
    const originW = win.width;
    const originH = win.height;
    const onMove = (move: PointerEvent) => {
      presentation.resize(viewId, originW + (move.clientX - startX), originH + (move.clientY - startY));
    };
    const onUp = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp, { once: true });
  };

  const startMosaicEdge = (
    viewId: string,
    edge: 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw',
    event: React.PointerEvent
  ) => {
    if (!mosaic || event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    presentation.focus(viewId);
    let lastX = event.clientX;
    let lastY = event.clientY;
    const onMove = (move: PointerEvent) => {
      const dx = move.clientX - lastX;
      const dy = move.clientY - lastY;
      lastX = move.clientX;
      lastY = move.clientY;
      if (dx === 0 && dy === 0) return;
      presentation.resizeMosaic(viewId, edge, dx, dy);
    };
    const onUp = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp, { once: true });
  };

  const startSplitter = (
    splitter: { firstId: string; secondId: string; orientation: 'vertical' | 'horizontal' },
    event: React.PointerEvent
  ) => {
    if (event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    let last = splitter.orientation === 'vertical' ? event.clientX : event.clientY;
    const onMove = (move: PointerEvent) => {
      const current = splitter.orientation === 'vertical' ? move.clientX : move.clientY;
      const delta = current - last;
      last = current;
      if (delta === 0) return;
      presentation.resizeAdjacent(splitter.firstId, splitter.secondId, splitter.orientation, delta);
    };
    const onUp = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp, { once: true });
  };

  return (
    <div
      ref={hostRef}
      className="nx-desktop-workspace"
      data-tour="terminals-windows"
      data-presentation={mosaic ? 'mosaic' : 'desktop'}
    >
      {windows.map(({ surface, win }) => {
        if (!win || win.minimized) return null;
        const viewId = surfaceViewId(surface);
        const attention = surface.data?.hasAttention === 'true';
        const unread = surface.data?.unreadAttention === 'true';
        const attentionKind = surface.data?.attentionKind || '';
        const agentName = win.customTitle || surface.data?.agentName || surface.title.replace(/\s·\s.*$/, '');
        const statusSuffix = surface.data?.statusSuffix || '';
        const dynamicTitle = surface.data?.dynamicTitle || '';
        const providerLabel = surface.data?.providerLabel || '';
        const accent = win.accent || '';
        const icon = win.icon || '';
        const style: React.CSSProperties = {
          ...(win.maximized
            ? { inset: 8, zIndex: win.zIndex }
            : {
                left: win.x,
                top: win.y,
                width: win.width,
                height: win.height,
                zIndex: win.zIndex,
              }),
          ...(accent ? ({ ['--nx-window-accent' as string]: accent } as React.CSSProperties) : {}),
        };
        return (
          <section
            key={viewId}
            className="nx-desktop-window"
            data-view-id={viewId}
            data-maximized={win.maximized ? 'true' : 'false'}
            data-mosaic={mosaic ? 'true' : undefined}
            data-accented={accent ? 'true' : undefined}
            data-attention={attention ? 'true' : undefined}
            data-unread={unread ? 'true' : undefined}
            data-attention-kind={attentionKind || undefined}
            data-icon={icon || undefined}
            style={style}
            onPointerDownCapture={() => {
              presentation.focus(viewId);
              if (chromeMenuViewId && chromeMenuViewId !== viewId) setChromeMenuViewId(null);
            }}
          >
            <header
              className="nx-desktop-window__titlebar"
              data-accented={accent ? 'true' : undefined}
              onPointerDown={(event) => {
                setChromeMenuViewId(null);
                startMove(viewId, event);
              }}
            >
              <div className="nx-desktop-window__title">
                {icon && <span className="nx-desktop-window__icon" aria-hidden="true">{icon}</span>}
                {(attention || unread) && (
                  <span
                    className="nx-desktop-window__attention-dot"
                    data-kind={attentionKind || 'needs_user'}
                    data-unread={unread ? 'true' : undefined}
                    aria-label={unread ? 'unread attention' : 'attention'}
                  />
                )}
                <strong title={agentName}>{agentName}</strong>
                {providerLabel && <small className="nx-desktop-window__provider">{providerLabel}</small>}
                {dynamicTitle && (
                  <small className="nx-desktop-window__dynamic" title={dynamicTitle}>
                    {dynamicTitle}
                  </small>
                )}
                {!dynamicTitle && statusSuffix && (
                  <small className="nx-desktop-window__status" data-kind={attentionKind || undefined}>
                    {statusSuffix}
                  </small>
                )}
              </div>
              <div className="nx-desktop-window__actions" onPointerDown={(event) => event.stopPropagation()}>
                <WindowChromeMenu
                  viewId={viewId}
                  open={chromeMenuViewId === viewId}
                  onOpenChange={(next) => setChromeMenuViewId(next ? viewId : null)}
                  customTitle={win.customTitle || ''}
                  accent={win.accent || ''}
                  icon={win.icon || ''}
                  onPatch={(chrome) => presentation.patchChrome(viewId, chrome)}
                />
                <IconButton label={`Minimize ${agentName}`} onClick={() => presentation.minimize(viewId)}>
                  <Minus size={13} />
                </IconButton>
                {!mosaic && (
                  <IconButton
                    label={win.maximized ? `Restore ${agentName}` : `Maximize ${agentName}`}
                    onClick={() => presentation.maximize(viewId)}
                  >
                    {win.maximized ? <Minimize2 size={13} /> : <Maximize2 size={13} />}
                  </IconButton>
                )}
                {surface.closable !== false && (
                  <IconButton label={`Close ${agentName}`} onClick={() => closeSurface(surface)}>
                    <X size={13} />
                  </IconButton>
                )}
              </div>
            </header>
            <div className="nx-desktop-window__body">{renderSurface(surface)}</div>
            {!mosaic && !win.maximized && (
              <div
                className="nx-desktop-window__resize"
                aria-hidden="true"
                onPointerDown={(event) => startResize(viewId, event)}
              />
            )}
            {mosaic &&
              (
                [
                  ['n', 'ns-resize'],
                  ['s', 'ns-resize'],
                  ['e', 'ew-resize'],
                  ['w', 'ew-resize'],
                  ['ne', 'nesw-resize'],
                  ['nw', 'nwse-resize'],
                  ['se', 'nwse-resize'],
                  ['sw', 'nesw-resize'],
                ] as const
              ).map(([edge, cursor]) => (
                <div
                  key={edge}
                  className={`nx-desktop-window__grip nx-desktop-window__grip--${edge}`}
                  style={{ cursor }}
                  aria-hidden="true"
                  onPointerDown={(event) => startMosaicEdge(viewId, edge, event)}
                />
              ))}
          </section>
        );
      })}
      {mosaic &&
        splitters.map((splitter) => {
          const style: React.CSSProperties =
            splitter.orientation === 'vertical'
              ? {
                  left: splitter.x - 3,
                  top: splitter.y,
                  width: 6,
                  height: splitter.length,
                  cursor: 'col-resize',
                }
              : {
                  left: splitter.x,
                  top: splitter.y - 3,
                  width: splitter.length,
                  height: 6,
                  cursor: 'row-resize',
                };
          return (
            <div
              key={splitter.id}
              className="nx-desktop-mosaic-splitter"
              data-orientation={splitter.orientation}
              style={style}
              onPointerDown={(event) => startSplitter(splitter, event)}
            />
          );
        })}
      <div className="nx-desktop-dock" aria-label="Minimized terminals">
        {windows
          .filter(({ win }) => win?.minimized)
          .map(({ surface }) => {
            const viewId = surfaceViewId(surface);
            const unread = surface.data?.unreadAttention === 'true';
            const label = surface.data?.agentName || surface.title;
            return (
              <button
                type="button"
                key={viewId}
                data-unread={unread ? 'true' : undefined}
                onClick={() => {
                  presentation.minimize(viewId);
                  presentation.focus(viewId);
                }}
              >
                {unread && <span className="nx-desktop-window__attention-dot" data-unread="true" />}
                {label}
              </button>
            );
          })}
      </div>
    </div>
  );
};
