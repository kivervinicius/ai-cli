import { describe, expect, it } from 'vitest';
import { parseRouteLocation, buildProjectRoute, buildPopoutRoute } from './routes';

describe('Router Integration Model', () => {
  it('correctly maps URL history and params', () => {
    const route = parseRouteLocation('/p/proj-alpha/missions/run-101');
    expect(route).toEqual({
      kind: 'project',
      projectId: 'proj-alpha',
      surface: 'missions',
      subId: 'run-101',
    });
  });

  it('generates proper project URLs for navigation without losing sub-path', () => {
    const url = buildProjectRoute('p1', 'work', 'session-123');
    expect(url).toBe('/p/p1/work/session-123');
  });

  it('generates proper popout URLs', () => {
    const url = buildPopoutRoute('p1', 'terminals');
    expect(url).toBe('/p/p1/popout/terminals');
  });
});
