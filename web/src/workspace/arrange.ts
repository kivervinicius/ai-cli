export type ArrangeBounds = { x: number; y: number; width: number; height: number };

export type TileRect = { viewId: string; x: number; y: number; width: number; height: number };

const GAP = 8;
const MIN_W = 280;
const MIN_H = 180;

function roundRect(rect: Omit<TileRect, 'viewId'> & { viewId: string }): TileRect {
  return {
    viewId: rect.viewId,
    x: Math.round(rect.x),
    y: Math.round(rect.y),
    width: Math.max(MIN_W, Math.round(rect.width)),
    height: Math.max(MIN_H, Math.round(rect.height)),
  };
}

function fill(bounds: ArrangeBounds, viewIds: string[]): TileRect[] {
  if (viewIds.length === 0) return [];
  return [
    roundRect({
      viewId: viewIds[0],
      x: bounds.x,
      y: bounds.y,
      width: bounds.width,
      height: bounds.height,
    }),
  ];
}

function columns(bounds: ArrangeBounds, viewIds: string[]): TileRect[] {
  const n = viewIds.length;
  if (n === 0) return [];
  const totalGap = GAP * (n - 1);
  const cellW = (bounds.width - totalGap) / n;
  return viewIds.map((viewId, index) =>
    roundRect({
      viewId,
      x: bounds.x + index * (cellW + GAP),
      y: bounds.y,
      width: cellW,
      height: bounds.height,
    })
  );
}

function rows(bounds: ArrangeBounds, viewIds: string[]): TileRect[] {
  const n = viewIds.length;
  if (n === 0) return [];
  const totalGap = GAP * (n - 1);
  const cellH = (bounds.height - totalGap) / n;
  return viewIds.map((viewId, index) =>
    roundRect({
      viewId,
      x: bounds.x,
      y: bounds.y + index * (cellH + GAP),
      width: bounds.width,
      height: cellH,
    })
  );
}

function grid2x2(bounds: ArrangeBounds, viewIds: string[]): TileRect[] {
  const ids = viewIds.slice(0, 4);
  const halfW = (bounds.width - GAP) / 2;
  const halfH = (bounds.height - GAP) / 2;
  const positions = [
    { x: bounds.x, y: bounds.y },
    { x: bounds.x + halfW + GAP, y: bounds.y },
    { x: bounds.x, y: bounds.y + halfH + GAP },
    { x: bounds.x + halfW + GAP, y: bounds.y + halfH + GAP },
  ];
  return ids.map((viewId, index) =>
    roundRect({
      viewId,
      x: positions[index].x,
      y: positions[index].y,
      width: halfW,
      height: halfH,
    })
  );
}

function masterStack(bounds: ArrangeBounds, viewIds: string[]): TileRect[] {
  if (viewIds.length < 3) return columns(bounds, viewIds);
  const masterW = (bounds.width - GAP) / 2;
  const stackIds = viewIds.slice(1);
  const master = roundRect({
    viewId: viewIds[0],
    x: bounds.x,
    y: bounds.y,
    width: masterW,
    height: bounds.height,
  });
  const stackBounds: ArrangeBounds = {
    x: bounds.x + masterW + GAP,
    y: bounds.y,
    width: masterW,
    height: bounds.height,
  };
  return [master, ...rows(stackBounds, stackIds)];
}

function rowPairs(bounds: ArrangeBounds, viewIds: string[]): TileRect[] {
  const n = viewIds.length;
  if (n === 0) return [];
  const rowCount = Math.ceil(n / 2);
  const totalGap = GAP * (rowCount - 1);
  const rowH = (bounds.height - totalGap) / rowCount;
  const tiles: TileRect[] = [];
  for (let row = 0; row < rowCount; row += 1) {
    const start = row * 2;
    const rowIds = viewIds.slice(start, start + 2);
    const rowBounds: ArrangeBounds = {
      x: bounds.x,
      y: bounds.y + row * (rowH + GAP),
      width: bounds.width,
      height: rowH,
    };
    tiles.push(...(rowIds.length === 1 ? fill(rowBounds, rowIds) : columns(rowBounds, rowIds)));
  }
  return tiles;
}

/**
 * Smart mosaic for Desktop v1:
 * 1 = fill, 2 = columns, 3 = 50% + stacked, 4 = 2×2, 5+ = rows of 2.
 */
export function arrangeSmart(bounds: ArrangeBounds, viewIds: string[]): TileRect[] {
  const ids = viewIds.filter(Boolean);
  if (ids.length === 0) return [];
  if (bounds.width < MIN_W || bounds.height < MIN_H) {
    return ids.map((viewId) =>
      roundRect({
        viewId,
        x: bounds.x,
        y: bounds.y,
        width: Math.max(MIN_W, bounds.width),
        height: Math.max(MIN_H, bounds.height),
      })
    );
  }
  switch (ids.length) {
    case 1:
      return fill(bounds, ids);
    case 2:
      return columns(bounds, ids);
    case 3:
      return masterStack(bounds, ids);
    case 4:
      return grid2x2(bounds, ids);
    default:
      return rowPairs(bounds, ids);
  }
}

export function scaleTilesToCanvas(
  tiles: Array<{ viewId: string; x: number; y: number; width: number; height: number }>,
  from: ArrangeBounds,
  to: ArrangeBounds
): TileRect[] {
  if (from.width <= 0 || from.height <= 0) return arrangeSmart(to, tiles.map((t) => t.viewId));
  const sx = to.width / from.width;
  const sy = to.height / from.height;
  return tiles.map((tile) =>
    roundRect({
      viewId: tile.viewId,
      x: to.x + (tile.x - from.x) * sx,
      y: to.y + (tile.y - from.y) * sy,
      width: tile.width * sx,
      height: tile.height * sy,
    })
  );
}

export const ARRANGE_GAP = GAP;
export const ARRANGE_MIN_W = MIN_W;
export const ARRANGE_MIN_H = MIN_H;

export type TileSplitter = {
  id: string;
  orientation: 'vertical' | 'horizontal';
  /** Grip center along the shared gap. */
  x: number;
  y: number;
  length: number;
  firstId: string;
  secondId: string;
};

type RectLike = { viewId: string; x: number; y: number; width: number; height: number };

function rangesOverlap(a0: number, a1: number, b0: number, b1: number, slack = 4): boolean {
  return a0 < b1 - slack && b0 < a1 - slack;
}

/**
 * Detect shared edges between tiles (Smart mosaic or user-adjusted).
 * Vertical splitter: first = left, second = right.
 * Horizontal splitter: first = top, second = bottom.
 */
export function findTileSplitters(tiles: RectLike[], gap = GAP): TileSplitter[] {
  const splitters: TileSplitter[] = [];
  const seen = new Set<string>();
  for (let i = 0; i < tiles.length; i += 1) {
    for (let j = 0; j < tiles.length; j += 1) {
      if (i === j) continue;
      const a = tiles[i];
      const b = tiles[j];
      const aRight = a.x + a.width;
      const aBottom = a.y + a.height;
      const gapSlack = gap + 6;

      // Vertical: A left of B, right edge near B.left
      if (
        Math.abs(aRight + gap - b.x) <= gapSlack &&
        rangesOverlap(a.y, aBottom, b.y, b.y + b.height)
      ) {
        const key = `v:${a.viewId}|${b.viewId}`;
        if (!seen.has(key)) {
          seen.add(key);
          const top = Math.max(a.y, b.y);
          const bottom = Math.min(aBottom, b.y + b.height);
          splitters.push({
            id: key,
            orientation: 'vertical',
            x: aRight + gap / 2,
            y: top,
            length: Math.max(24, bottom - top),
            firstId: a.viewId,
            secondId: b.viewId,
          });
        }
      }

      // Horizontal: A above B
      if (
        Math.abs(aBottom + gap - b.y) <= gapSlack &&
        rangesOverlap(a.x, aRight, b.x, b.x + b.width)
      ) {
        const key = `h:${a.viewId}|${b.viewId}`;
        if (!seen.has(key)) {
          seen.add(key);
          const left = Math.max(a.x, b.x);
          const right = Math.min(aRight, b.x + b.width);
          splitters.push({
            id: key,
            orientation: 'horizontal',
            x: left,
            y: aBottom + gap / 2,
            length: Math.max(24, right - left),
            firstId: a.viewId,
            secondId: b.viewId,
          });
        }
      }
    }
  }
  return splitters;
}

/**
 * Move the shared edge by `delta` pixels (positive grows first / shrinks second).
 * Returns patched rects for the two tiles, or null if clamp would violate mins.
 */
export function applySharedEdgeDelta(
  first: RectLike,
  second: RectLike,
  orientation: 'vertical' | 'horizontal',
  delta: number
): { first: RectLike; second: RectLike } | null {
  const vertical = orientation === 'vertical';
  const leading = vertical
    ? first.x <= second.x
      ? first
      : second
    : first.y <= second.y
      ? first
      : second;
  const trailing = leading.viewId === first.viewId ? second : first;
  if (vertical) {
    const nextLeadingW = leading.width + delta;
    const nextTrailingW = trailing.width - delta;
    if (nextLeadingW < MIN_W || nextTrailingW < MIN_W) return null;
    const nextLeading = { ...leading, width: Math.round(nextLeadingW) };
    const nextTrailing = {
      ...trailing,
      x: Math.round(trailing.x + delta),
      width: Math.round(nextTrailingW),
    };
    return leading.viewId === first.viewId
      ? { first: nextLeading, second: nextTrailing }
      : { first: nextTrailing, second: nextLeading };
  }
  const nextLeadingH = leading.height + delta;
  const nextTrailingH = trailing.height - delta;
  if (nextLeadingH < MIN_H || nextTrailingH < MIN_H) return null;
  const nextLeading = { ...leading, height: Math.round(nextLeadingH) };
  const nextTrailing = {
    ...trailing,
    y: Math.round(trailing.y + delta),
    height: Math.round(nextTrailingH),
  };
  return leading.viewId === first.viewId
    ? { first: nextLeading, second: nextTrailing }
    : { first: nextTrailing, second: nextLeading };
}
