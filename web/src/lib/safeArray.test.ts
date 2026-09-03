import { describe, expect, it } from 'vitest';
import { asArray, asStringArray } from './safeArray';

describe('asArray', () => {
  it('returns the same array reference when already an array', () => {
    const input = ['a'];
    expect(asArray(input)).toBe(input);
  });

  it('coerces null/undefined/non-arrays to empty arrays', () => {
    expect(asArray(null)).toEqual([]);
    expect(asArray(undefined)).toEqual([]);
    expect(asArray('x')).toEqual([]);
    expect(asArray({ length: 2 })).toEqual([]);
  });
});

describe('asStringArray', () => {
  it('keeps only string items', () => {
    expect(asStringArray(['a', 1, null, 'b'])).toEqual(['a', 'b']);
    expect(asStringArray(null)).toEqual([]);
  });
});
