/** Hostnames where WebSocket query-token auth is allowed (matches Go isLoopbackHost). */
export function isLoopbackHostname(host: string): boolean {
  const hostName = host.split(':')[0]?.trim().toLowerCase() ?? '';
  if (!hostName) return false;
  if (hostName === 'localhost' || hostName === 'nexus.local') return true;
  if (hostName === '127.0.0.1' || hostName === '::1') return true;
  // IPv4 loopback range 127.0.0.0/8
  if (/^127(?:\.\d{1,3}){3}$/.test(hostName)) return true;
  return false;
}

/**
 * Session IDs must not appear in WebSocket URLs outside loopback.
 * Tunnel and private --remote binds authenticate via HttpOnly cookie only.
 */
export function shouldAttachWebSocketQueryToken(host: string): boolean {
  const hostName = host.split(':')[0]?.toLowerCase() ?? '';
  if (hostName.endsWith('.trycloudflare.com')) return false;
  return isLoopbackHostname(host);
}
