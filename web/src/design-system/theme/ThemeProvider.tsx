import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import {
  defaultThemePreferences,
  normalizeThemePreferences,
  resolveScheme,
  themeStorageKey,
  type ThemeAccent,
  type ThemeDensity,
  type ThemePreferences,
  type ThemeScheme,
} from './theme';

interface ThemeContextValue extends ThemePreferences {
  resolvedScheme: 'dark' | 'light' | 'high-contrast';
  setScheme: (scheme: ThemeScheme) => void;
  setAccent: (accent: ThemeAccent) => void;
  setDensity: (density: ThemeDensity) => void;
  setReducedMotion: (value: boolean) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

function loadPreferences(): ThemePreferences {
  if (typeof window === 'undefined') return defaultThemePreferences;
  try {
    return normalizeThemePreferences(JSON.parse(window.localStorage.getItem(themeStorageKey) || 'null'));
  } catch {
    return defaultThemePreferences;
  }
}

export const ThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [preferences, setPreferences] = useState<ThemePreferences>(loadPreferences);
  const [systemDark, setSystemDark] = useState(() => typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches);

  useEffect(() => {
    const media = window.matchMedia?.('(prefers-color-scheme: dark)');
    if (!media) return;
    const update = () => setSystemDark(media.matches);
    media.addEventListener?.('change', update);
    return () => media.removeEventListener?.('change', update);
  }, []);

  const resolvedScheme = resolveScheme(preferences.scheme, systemDark);

  useEffect(() => {
    window.localStorage.setItem(themeStorageKey, JSON.stringify(preferences));
    const root = document.documentElement;
    root.dataset.theme = resolvedScheme;
    root.dataset.accent = preferences.accent;
    root.dataset.density = preferences.density;
    root.dataset.reducedMotion = preferences.reducedMotion ? 'true' : 'false';
    root.style.colorScheme = resolvedScheme === 'light' ? 'light' : 'dark';
  }, [preferences, resolvedScheme]);

  const value = useMemo<ThemeContextValue>(() => ({
    ...preferences,
    resolvedScheme,
    setScheme: (scheme) => setPreferences((current) => ({ ...current, scheme })),
    setAccent: (accent) => setPreferences((current) => ({ ...current, accent })),
    setDensity: (density) => setPreferences((current) => ({ ...current, density })),
    setReducedMotion: (reducedMotion) => setPreferences((current) => ({ ...current, reducedMotion })),
  }), [preferences, resolvedScheme]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
};

export function useTheme(): ThemeContextValue {
  const value = useContext(ThemeContext);
  if (!value) throw new Error('useTheme must be used inside ThemeProvider');
  return value;
}
