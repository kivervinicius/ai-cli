import { describe, expect, it } from 'vitest';
import { normalizeThemePreferences, resolveScheme, themeStorageKey, type ThemePreferences } from './theme';

describe('theme model', () => {
  it('resolves system scheme from media preference', () => {
    expect(resolveScheme('system', true)).toBe('dark');
    expect(resolveScheme('system', false)).toBe('light');
  });

  it('keeps explicit and high contrast schemes', () => {
    expect(resolveScheme('dark', false)).toBe('dark');
    expect(resolveScheme('high-contrast', true)).toBe('high-contrast');
  });

  it('normalizes invalid persisted values', () => {
    const value = normalizeThemePreferences({ scheme: 'pink', accent: 'orange', density: 'huge' } as unknown as ThemePreferences);
    expect(value).toEqual({ scheme: 'system', accent: 'purple', density: 'compact', reducedMotion: false });
  });

  it('uses a stable storage key', () => expect(themeStorageKey).toBe('iapro:nexus:theme:v1'));
});
