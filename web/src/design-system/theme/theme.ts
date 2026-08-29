export type ThemeScheme = 'system' | 'dark' | 'light' | 'high-contrast';
export type ThemeAccent = 'purple' | 'blue' | 'cyan' | 'neutral';
export type ThemeDensity = 'compact' | 'comfortable';

export interface ThemePreferences {
  scheme: ThemeScheme;
  accent: ThemeAccent;
  density: ThemeDensity;
  reducedMotion: boolean;
}

export const themeStorageKey = 'iapro:nexus:theme:v1';
export const defaultThemePreferences: ThemePreferences = {
  scheme: 'system',
  accent: 'purple',
  density: 'compact',
  reducedMotion: false,
};

export function resolveScheme(scheme: ThemeScheme, systemDark: boolean): Exclude<ThemeScheme, 'system'> {
  return scheme === 'system' ? (systemDark ? 'dark' : 'light') : scheme;
}

export function normalizeThemePreferences(input?: Partial<ThemePreferences> | null): ThemePreferences {
  const schemes: ThemeScheme[] = ['system', 'dark', 'light', 'high-contrast'];
  const accents: ThemeAccent[] = ['purple', 'blue', 'cyan', 'neutral'];
  const densities: ThemeDensity[] = ['compact', 'comfortable'];
  return {
    scheme: schemes.includes(input?.scheme as ThemeScheme) ? input!.scheme! : defaultThemePreferences.scheme,
    accent: accents.includes(input?.accent as ThemeAccent) ? input!.accent! : defaultThemePreferences.accent,
    density: densities.includes(input?.density as ThemeDensity) ? input!.density! : defaultThemePreferences.density,
    reducedMotion: Boolean(input?.reducedMotion),
  };
}
