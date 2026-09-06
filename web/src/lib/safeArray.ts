/**
 * JSON from Go often encodes empty slices as `null`. Treat unknown values as [].
 * Use this at API boundaries and before `.map` / `.length` / `.filter`.
 */
export function asArray<T = unknown>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}

export function asStringArray(value: unknown): string[] {
  return asArray(value).filter((item): item is string => typeof item === 'string');
}
