export type BrowserSessionState = 'loading' | 'ready' | 'unauthenticated';

export function resolveSessionState(session: { authenticated: boolean; csrf_token?: string }): BrowserSessionState {
  return session.authenticated ? 'ready' : 'unauthenticated';
}
