import { describe, expect, it } from 'vitest';
import { clampMenuPosition } from './ContextMenu';

describe('clampMenuPosition', () => {
  it('keeps the menu inside the viewport', () => {
    expect(clampMenuPosition(10, 10, 200, 120, 800, 600)).toEqual({ x: 10, y: 10 });
    expect(clampMenuPosition(780, 580, 200, 120, 800, 600)).toEqual({ x: 592, y: 472 });
  });
});
