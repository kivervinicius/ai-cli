import React, { createContext, useContext, useEffect, useMemo, useReducer } from 'react';
import {
  createPresentationState,
  focusDesktopWindow,
  migratePresentationState,
  moveDesktopWindow,
  rearrangeSmart,
  resizeAdjacentDesktopWindows,
  resizeDesktopWindow,
  setActivePtyView,
  setPresentationCanvas,
  setPresentationMode,
  syncDesktopWindows,
  toggleDesktopMaximize,
  toggleDesktopMinimize,
  type WorkspacePresentationMode,
  type WorkspacePresentationState,
} from './presentation';
import type { ArrangeBounds } from './arrange';
import type { WorkspaceSurface } from './model';

const storageKey = (projectId: string) => `iapro:nexus:workspace:${projectId}:presentation:v1`;

interface ContextValue {
  state: WorkspacePresentationState;
  setMode: (mode: WorkspacePresentationMode) => void;
  sync: (surfaces: WorkspaceSurface[]) => void;
  focus: (viewId: string) => void;
  setActivePty: (viewId: string) => void;
  move: (viewId: string, x: number, y: number) => void;
  resize: (viewId: string, width: number, height: number) => void;
  resizeAdjacent: (
    firstId: string,
    secondId: string,
    orientation: 'vertical' | 'horizontal',
    delta: number
  ) => void;
  minimize: (viewId: string) => void;
  maximize: (viewId: string) => void;
  setCanvas: (canvas: ArrangeBounds) => void;
  rearrange: () => void;
}

type Action =
  | { type: 'mode'; mode: WorkspacePresentationMode }
  | { type: 'sync'; surfaces: WorkspaceSurface[] }
  | { type: 'focus'; viewId: string }
  | { type: 'setActivePty'; viewId: string }
  | { type: 'move'; viewId: string; x: number; y: number }
  | { type: 'resize'; viewId: string; width: number; height: number }
  | {
      type: 'resizeAdjacent';
      firstId: string;
      secondId: string;
      orientation: 'vertical' | 'horizontal';
      delta: number;
    }
  | { type: 'minimize'; viewId: string }
  | { type: 'maximize'; viewId: string }
  | { type: 'canvas'; canvas: ArrangeBounds }
  | { type: 'rearrange' };

function reducer(state: WorkspacePresentationState, action: Action): WorkspacePresentationState {
  switch (action.type) {
    case 'mode': return setPresentationMode(state, action.mode);
    case 'sync': return syncDesktopWindows(state, action.surfaces);
    case 'focus': return focusDesktopWindow(state, action.viewId);
    case 'setActivePty': return setActivePtyView(state, action.viewId);
    case 'move': return moveDesktopWindow(state, action.viewId, action.x, action.y);
    case 'resize': return resizeDesktopWindow(state, action.viewId, action.width, action.height);
    case 'resizeAdjacent':
      return resizeAdjacentDesktopWindows(
        state,
        action.firstId,
        action.secondId,
        action.orientation,
        action.delta
      );
    case 'minimize': return toggleDesktopMinimize(state, action.viewId);
    case 'maximize': return toggleDesktopMaximize(state, action.viewId);
    case 'canvas': return setPresentationCanvas(state, action.canvas);
    case 'rearrange': return rearrangeSmart(state);
  }
}

function load(projectId: string): WorkspacePresentationState {
  try {
    const raw = window.localStorage.getItem(storageKey(projectId));
    if (!raw) return createPresentationState();
    return migratePresentationState(JSON.parse(raw));
  } catch {
    return createPresentationState();
  }
}

const WorkspacePresentationContext = createContext<ContextValue | null>(null);

export const WorkspacePresentationProvider: React.FC<{ projectId: string; children: React.ReactNode }> = ({ projectId, children }) => {
  const [state, dispatch] = useReducer(reducer, projectId, load);
  useEffect(() => { window.localStorage.setItem(storageKey(projectId), JSON.stringify(state)); }, [projectId, state]);
  const value = useMemo<ContextValue>(() => ({
    state,
    setMode: (mode) => dispatch({ type: 'mode', mode }),
    sync: (surfaces) => dispatch({ type: 'sync', surfaces }),
    focus: (viewId) => dispatch({ type: 'focus', viewId }),
    setActivePty: (viewId) => dispatch({ type: 'setActivePty', viewId }),
    move: (viewId, x, y) => dispatch({ type: 'move', viewId, x, y }),
    resize: (viewId, width, height) => dispatch({ type: 'resize', viewId, width, height }),
    resizeAdjacent: (firstId, secondId, orientation, delta) =>
      dispatch({ type: 'resizeAdjacent', firstId, secondId, orientation, delta }),
    minimize: (viewId) => dispatch({ type: 'minimize', viewId }),
    maximize: (viewId) => dispatch({ type: 'maximize', viewId }),
    setCanvas: (canvas) => dispatch({ type: 'canvas', canvas }),
    rearrange: () => dispatch({ type: 'rearrange' }),
  }), [state]);
  return <WorkspacePresentationContext.Provider value={value}>{children}</WorkspacePresentationContext.Provider>;
};

export function useWorkspacePresentation(): ContextValue {
  const value = useContext(WorkspacePresentationContext);
  if (!value) throw new Error('useWorkspacePresentation must be used inside WorkspacePresentationProvider');
  return value;
}
