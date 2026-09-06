import { PlatformBridge, setPlatformBridge, getPlatformBridge } from './platformBridge';
import { WebBridge } from './webBridge';
import { DesktopBridge } from './desktopBridge';

export * from './capabilities';
export * from './platformBridge';
export * from './webBridge';
export * from './desktopBridge';

export function isDesktopApp(): boolean {
  return typeof window !== 'undefined' && (!!window.go || !!window.runtime);
}

export function initPlatformBridge(): PlatformBridge {
  const bridge = isDesktopApp() ? new DesktopBridge() : new WebBridge();
  setPlatformBridge(bridge);
  return bridge;
}
