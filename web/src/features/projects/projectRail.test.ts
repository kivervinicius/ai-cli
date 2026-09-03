// @ts-expect-error Vitest executes in Node; browser tsconfig intentionally omits @types/node.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./ProjectRail.tsx', import.meta.url), 'utf8');

describe('Project rail Maestro entry', () => {
  it('does not expose Maestro as a workspace surface', () => {
    expect(source).not.toContain("rail.maestroMethod");
    expect(source).not.toContain("id: 'maestro'");
  });

  it('keeps the project overview available from the project rail', () => {
    expect(source).toContain("id: 'overview'");
    expect(source).toContain("label: t('nav.overview')");
  });
});
