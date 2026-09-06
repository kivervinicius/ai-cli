import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { PtyLiveChrome } from './ptyLiveChrome';

const emptyChrome: PtyLiveChrome = { title: '', questionnaire: false };

interface PtyLiveChromeContextValue {
  byViewId: Record<string, PtyLiveChrome>;
  setLive: (viewId: string, patch: Partial<PtyLiveChrome>) => void;
  clearLive: (viewId: string) => void;
}

const PtyLiveChromeContext = createContext<PtyLiveChromeContextValue | null>(null);

/**
 * Coalesce settitle / questionnaire updates so AGY/Codex OSC spam does not
 * re-render every mosaic window on each chunk (layout thrash → resize storms).
 */
export const PtyLiveChromeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [byViewId, setByViewId] = useState<Record<string, PtyLiveChrome>>({});
  const publishedRef = useRef<Record<string, PtyLiveChrome>>({});
  const pendingRef = useRef<Record<string, PtyLiveChrome>>({});
  const flushTimerRef = useRef<number | undefined>(undefined);

  const flush = useCallback(() => {
    flushTimerRef.current = undefined;
    const pending = pendingRef.current;
    pendingRef.current = {};
    const keys = Object.keys(pending);
    if (keys.length === 0) return;
    setByViewId((current) => {
      let changed = false;
      const next = { ...current };
      for (const key of keys) {
        const value = pending[key];
        const prev = current[key] || emptyChrome;
        if (value.title === prev.title && value.questionnaire === prev.questionnaire) continue;
        next[key] = value;
        changed = true;
      }
      if (!changed) return current;
      publishedRef.current = next;
      return next;
    });
  }, []);

  const setLive = useCallback(
    (viewId: string, patch: Partial<PtyLiveChrome>) => {
      const key = viewId.trim();
      if (!key) return;
      const published = publishedRef.current[key] || emptyChrome;
      const staged = pendingRef.current[key] || published;
      const next: PtyLiveChrome = {
        title: patch.title !== undefined ? patch.title : staged.title,
        questionnaire:
          patch.questionnaire !== undefined ? patch.questionnaire : staged.questionnaire,
      };
      if (next.title === staged.title && next.questionnaire === staged.questionnaire) return;
      pendingRef.current[key] = next;
      if (flushTimerRef.current === undefined) {
        flushTimerRef.current = window.setTimeout(flush, 80);
      }
    },
    [flush],
  );

  const clearLive = useCallback((viewId: string) => {
    const key = viewId.trim();
    if (!key) return;
    delete pendingRef.current[key];
    setByViewId((current) => {
      if (!(key in current)) return current;
      const { [key]: _removed, ...rest } = current;
      publishedRef.current = rest;
      return rest;
    });
  }, []);

  useEffect(() => {
    return () => {
      if (flushTimerRef.current !== undefined) window.clearTimeout(flushTimerRef.current);
    };
  }, []);

  const value = useMemo(() => ({ byViewId, setLive, clearLive }), [byViewId, setLive, clearLive]);
  return <PtyLiveChromeContext.Provider value={value}>{children}</PtyLiveChromeContext.Provider>;
};

export function usePtyLiveChrome(): PtyLiveChromeContextValue {
  const value = useContext(PtyLiveChromeContext);
  if (!value) throw new Error('usePtyLiveChrome must be used inside PtyLiveChromeProvider');
  return value;
}

export function usePtyLiveChromeOptional(): PtyLiveChromeContextValue | null {
  return useContext(PtyLiveChromeContext);
}

export function liveChromeFor(
  byViewId: Record<string, PtyLiveChrome>,
  viewId: string,
): PtyLiveChrome {
  return byViewId[viewId] || emptyChrome;
}
