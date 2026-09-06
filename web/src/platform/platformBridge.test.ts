import { describe, it, expect, beforeEach } from 'vitest';
import { WebBridge } from './webBridge';
import { DesktopBridge } from './desktopBridge';
import { initPlatformBridge, getPlatformBridge } from './index';

describe('PlatformBridge', () => {
  beforeEach(() => {
    (globalThis as any).window = globalThis;
    delete (globalThis as any).window.go;
    delete (globalThis as any).window.runtime;
  });

  it('initializes WebBridge in standard browser environment', () => {
    const bridge = initPlatformBridge();
    expect(bridge.kind).toBe('web');
    expect(bridge.getCapabilities().native).toBe(false);
    expect(bridge.getCapabilities().filePicker).toBe(false);
    expect(bridge.getCapabilities().folderPicker).toBe(false);
    expect(getPlatformBridge()).toBe(bridge);
  });

  it('initializes DesktopBridge when Wails bindings are detected', () => {
    (globalThis as any).window.go = {
      desktop: {
        App: {
          SelectDirectory: async () => '/home/user/project',
        },
      },
    };
    const bridge = initPlatformBridge();
    expect(bridge.kind).toBe('desktop');
    expect(bridge.getCapabilities().native).toBe(true);
    expect(bridge.getCapabilities().filePicker).toBe(true);
    expect(bridge.getCapabilities().folderPicker).toBe(true);
    expect(bridge.getCapabilities().tray).toBe(true);
  });

  it('selectDirectory delegates to desktop App binding', async () => {
    (globalThis as any).window.go = {
      desktop: {
        App: {
          SelectDirectory: async (title?: string) => `/selected/path/${title || 'default'}`,
        },
      },
    };
    const bridge = new DesktopBridge();
    const result = await bridge.selectDirectory('Choose folder');
    expect(result).toBe('/selected/path/Choose folder');
  });

  it('selectDirectory returns null in WebBridge', async () => {
    const bridge = new WebBridge();
    const result = await bridge.selectDirectory();
    expect(result).toBeNull();
  });
});
