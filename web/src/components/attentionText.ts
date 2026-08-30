/**
 * Runtime attention text can contain replacement characters when a provider
 * emits bytes that are not valid UTF-8. Do not expose that transport artifact
 * in the shell; use a stable fallback instead.
 */
export function sanitizeAttentionText(value: unknown, fallback: string): string {
  if (typeof value !== 'string') return fallback;

  const sanitized = value
    .replace(/\uFFFD+/g, ' ')
    // eslint-disable-next-line no-control-regex -- provider transport noise
    .replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();

  return sanitized || fallback;
}
