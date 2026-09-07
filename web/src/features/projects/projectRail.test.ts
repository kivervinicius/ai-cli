import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./ProjectRail.tsx', import.meta.url), 'utf8');

describe('Project rail Maestro entry', () => {
  it('exposes Maestro as a system surface in optional tools', () => {
    expect(source).not.toContain('rail.maestroMethod');
    expect(source).toContain("id: 'maestro'");
  });

  it('keeps the project overview available from the project rail', () => {
    expect(source).toContain("id: 'overview'");
    expect(source).toContain("label: t('nav.overview')");
  });
});
