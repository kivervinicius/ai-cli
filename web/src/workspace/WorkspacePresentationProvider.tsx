import React, { createContext, useContext, useEffect, useMemo, useReducer } from 'react';
import {
  createPresentationState,
  focusDesktopWindow,
  moveDesktopWindow,
  resizeDesktopWindow,
  syncDesktopWindows,
  toggleDesktopMaximize,
  toggleDesktopMinimize,
  type WorkspacePresentationMode,
  type WorkspacePresentationState,
} from './presentation';
import type { WorkspaceSurface } from './model';

const storageKey = (projectId: string) => `iapro:nexus:workspace:${projectId}:presentation:v1`;

interface ContextValue {
  state: WorkspacePresentationState;
  setMode: (mode: WorkspacePresentationMode) => void;
  sync: (surfaces: WorkspaceSurface[]) => void;
  focus: (viewId: string) => void;
  move: (viewId: string, x: number, y: number) => void;
  resize: (viewId: string, width: number, height: number) => void;
  minimize: (viewId: string) => void;
  maximize: (viewId: string) => void;
}

type Action =
  | { type: 'mode'; mode: WorkspacePresentationMode }
  | { type: 'sync'; surfaces: WorkspaceSurface[] }
  | { type: 'focus'; viewId: string }
  | { type: 'move'; viewId: string; x: number; y: number }
  | { type: 'resize'; viewId: string; width: number; height: number }
  | { type: 'minimize'; viewId: string }
  | { type: 'maximize'; viewId: string };

function reducer(state: WorkspacePresentationState, action: Action): WorkspacePresentationState {
  switch (action.type) {
    case 'mode': return { ...state, mode: action.mode };
    case 'sync': return syncDesktopWindows(state, action.surfaces);
    case 'focus': return focusDesktopWindow(state, action.viewId);
    case 'move': return moveDesktopWindow(state, action.viewId, action.x, action.y);
    case 'resize': return resizeDesktopWindow(state, action.viewId, action.width, action.height);
    case 'minimize': return toggleDesktopMinimize(state, action.viewId);
    case 'maximize': return toggleDesktopMaximize(state, action.viewId);
  }
}

function load(projectId: string): WorkspacePresentationState {
  try {
    const raw = window.localStorage.getItem(storageKey(projectId));
    if (!raw) return createPresentationState();
    const parsed = JSON.parse(raw) as WorkspacePresentationState;
    return parsed?.version === 1 && (parsed.mode === 'TABS' || parsed.mode === 'DESKTOP') && parsed.windows ? parsed : createPresentationState();
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
    move: (viewId, x, y) => dispatch({ type: 'move', viewId, x, y }),
    resize: (viewId, width, height) => dispatch({ type: 'resize', viewId, width, height }),
    minimize: (viewId) => dispatch({ type: 'minimize', viewId }),
    maximize: (viewId) => dispatch({ type: 'maximize', viewId }),
  }), [state]);
  return <WorkspacePresentationContext.Provider value={value}>{children}</WorkspacePresentationContext.Provider>;
};

export function useWorkspacePresentation(): ContextValue {
  const value = useContext(WorkspacePresentationContext);
  if (!value) throw new Error('useWorkspacePresentation must be used inside WorkspacePresentationProvider');
  return value;
}
