import { describe, expect, it } from 'vitest';
import { applySharedEdgeDelta, arrangeSmart, findTileSplitters, scaleTilesToCanvas } from './arrange';

const bounds = { x: 8, y: 8, width: 1000, height: 600 };

describe('arrangeSmart', () => {
  it('fills a single window', () => {
    const tiles = arrangeSmart(bounds, ['a']);
    expect(tiles).toEqual([{ viewId: 'a', x: 8, y: 8, width: 1000, height: 600 }]);
  });

  it('splits two windows into equal columns', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b']);
    expect(tiles).toHaveLength(2);
    expect(tiles[0].width).toBe(tiles[1].width);
    expect(tiles[0].x + tiles[0].width + 8).toBe(tiles[1].x);
    expect(tiles[0].height).toBe(600);
  });

  it('uses master + stack for three windows', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b', 'c']);
    expect(tiles).toHaveLength(3);
    expect(tiles[0].height).toBe(600);
    expect(tiles[1].x).toBe(tiles[2].x);
    expect(tiles[1].y + tiles[1].height + 8).toBe(tiles[2].y);
  });

  it('uses a 2x2 grid for four windows', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b', 'c', 'd']);
    expect(tiles).toHaveLength(4);
    expect(new Set(tiles.map((t) => `${t.x},${t.y}`)).size).toBe(4);
  });

  it('packs five-plus windows into rows of two', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b', 'c', 'd', 'e']);
    expect(tiles).toHaveLength(5);
    const bottomRow = tiles.filter((t) => t.y > bounds.y + 100);
    expect(bottomRow.length).toBeGreaterThanOrEqual(1);
  });

  it('returns empty for no view ids', () => {
    expect(arrangeSmart(bounds, [])).toEqual([]);
  });
});

describe('findTileSplitters', () => {
  it('exposes a vertical splitter between two columns', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b']);
    const splitters = findTileSplitters(tiles);
    expect(splitters).toHaveLength(1);
    expect(splitters[0]).toMatchObject({ orientation: 'vertical', firstId: 'a', secondId: 'b' });
  });

  it('exposes vertical and horizontal splitters for master+stack', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b', 'c']);
    const splitters = findTileSplitters(tiles);
    expect(splitters.some((s) => s.orientation === 'vertical' && s.firstId === 'a')).toBe(true);
    expect(splitters.some((s) => s.orientation === 'horizontal' && s.firstId === 'b' && s.secondId === 'c')).toBe(true);
  });
});

describe('applySharedEdgeDelta', () => {
  it('grows the left tile and shrinks the right tile', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b']);
    const next = applySharedEdgeDelta(tiles[0], tiles[1], 'vertical', 40);
    expect(next).not.toBeNull();
    expect(next!.first.width).toBe(tiles[0].width + 40);
    expect(next!.second.width).toBe(tiles[1].width - 40);
    expect(next!.second.x).toBe(tiles[1].x + 40);
  });

  it('keeps growing the geometric left tile even if ids are reversed', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b']);
    const next = applySharedEdgeDelta(tiles[1], tiles[0], 'vertical', 40);
    expect(next).not.toBeNull();
    expect(next!.second.width).toBe(tiles[0].width + 40);
    expect(next!.first.width).toBe(tiles[1].width - 40);
    expect(next!.first.x).toBe(tiles[1].x + 40);
  });

  it('rejects deltas that violate min size', () => {
    const tiles = arrangeSmart(bounds, ['a', 'b']);
    expect(applySharedEdgeDelta(tiles[0], tiles[1], 'vertical', 900)).toBeNull();
  });
});

describe('scaleTilesToCanvas', () => {
  it('scales tile geometry when the canvas changes size', () => {
    const from = { x: 0, y: 0, width: 1000, height: 500 };
    const to = { x: 0, y: 0, width: 2000, height: 1000 };
    const tiles = arrangeSmart(from, ['a', 'b']);
    const scaled = scaleTilesToCanvas(tiles, from, to);
    expect(scaled[0].width).toBeGreaterThan(tiles[0].width);
    expect(scaled[0].height).toBeGreaterThan(tiles[1].height);
  });
});
