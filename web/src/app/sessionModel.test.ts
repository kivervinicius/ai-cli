import { describe, expect, it } from 'vitest';
import { resolveSessionState } from './sessionModel';

describe('session model', () => {
  it('does not enter the workspace when the browser session is unauthenticated', () => {
    expect(resolveSessionState({ authenticated: false })).toBe('unauthenticated');
  });

  it('enters the workspace only after an authenticated session is confirmed', () => {
    expect(resolveSessionState({ authenticated: true, csrf_token: 'csrf' })).toBe('ready');
  });
});
