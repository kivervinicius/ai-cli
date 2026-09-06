import { PlatformBridge } from './platformBridge';
import { PlatformCapabilities, FilePickerOptions, NotificationOptions } from './capabilities';

declare global {
  interface Window {
    go?: {
      desktop?: {
        App?: {
          SelectDirectory?: (title?: string) => Promise<string>;
          SelectFile?: (options?: FilePickerOptions) => Promise<string>;
          ShowNotification?: (options: NotificationOptions) => Promise<void>;
          OpenExternal?: (url: string) => Promise<void>;
          GetSystemTheme?: () => Promise<'light' | 'dark' | 'unknown'>;
          IsMaximized?: () => Promise<boolean>;
          MinimizeWindow?: () => Promise<void>;
          MaximizeWindow?: () => Promise<void>;
          UnmaximizeWindow?: () => Promise<void>;
          CloseWindow?: () => Promise<void>;
          QuitApp?: () => Promise<void>;
          GetCapabilities?: () => Promise<PlatformCapabilities>;
        };
      };
    };
    runtime?: {
      BrowserOpenURL?: (url: string) => void;
      Quit?: () => void;
      WindowMinimise?: () => void;
      WindowToggleMaximise?: () => void;
    };
  }
}

export class DesktopBridge implements PlatformBridge {
  readonly kind = 'desktop' as const;

  private get appBinding() {
    return window.go?.desktop?.App;
  }

  getCapabilities(): PlatformCapabilities {
    return {
      native: true,
      filePicker: true,
      folderPicker: true,
      notifications: true,
      tray: true,
      nativeMenus: true,
      deepLinks: true,
      autoStart: true,
      windowManagement: true,
    };
  }

  async selectDirectory(title?: string): Promise<string | null> {
    if (this.appBinding?.SelectDirectory) {
      try {
        const path = await this.appBinding.SelectDirectory(title);
        return path || null;
      } catch {
        return null;
      }
    }
    return null;
  }

  async selectFile(options?: FilePickerOptions): Promise<string | null> {
    if (this.appBinding?.SelectFile) {
      try {
        const path = await this.appBinding.SelectFile(options);
        return path || null;
      } catch {
        return null;
      }
    }
    return null;
  }

  async showNotification(options: NotificationOptions): Promise<void> {
    if (this.appBinding?.ShowNotification) {
      try {
        await this.appBinding.ShowNotification(options);
        return;
      } catch {
        // Fallback to web notification if available
      }
    }
    if (
      typeof window !== 'undefined' &&
      'Notification' in window &&
      Notification.permission === 'granted'
    ) {
      new Notification(options.title, { body: options.body, icon: options.icon });
    }
  }

  async openExternal(url: string): Promise<void> {
    if (this.appBinding?.OpenExternal) {
      try {
        await this.appBinding.OpenExternal(url);
        return;
      } catch {
        // Fallback
      }
    }
    if (window.runtime?.BrowserOpenURL) {
      window.runtime.BrowserOpenURL(url);
      return;
    }
    window.open(url, '_blank', 'noopener,noreferrer');
  }

  async getSystemTheme(): Promise<'light' | 'dark' | 'unknown'> {
    if (this.appBinding?.GetSystemTheme) {
      try {
        return await this.appBinding.GetSystemTheme();
      } catch {
        // Fallback
      }
    }
    if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
      return 'dark';
    }
    return 'light';
  }

  async isMaximized(): Promise<boolean> {
    if (this.appBinding?.IsMaximized) {
      return await this.appBinding.IsMaximized();
    }
    return false;
  }

  async minimizeWindow(): Promise<void> {
    if (this.appBinding?.MinimizeWindow) {
      await this.appBinding.MinimizeWindow();
    } else if (window.runtime?.WindowMinimise) {
      window.runtime.WindowMinimise();
    }
  }

  async maximizeWindow(): Promise<void> {
    if (this.appBinding?.MaximizeWindow) {
      await this.appBinding.MaximizeWindow();
    } else if (window.runtime?.WindowToggleMaximise) {
      window.runtime.WindowToggleMaximise();
    }
  }

  async unmaximizeWindow(): Promise<void> {
    if (this.appBinding?.UnmaximizeWindow) {
      await this.appBinding.UnmaximizeWindow();
    }
  }

  async closeWindow(): Promise<void> {
    if (this.appBinding?.CloseWindow) {
      await this.appBinding.CloseWindow();
    }
  }

  async quitApp(): Promise<void> {
    if (this.appBinding?.QuitApp) {
      await this.appBinding.QuitApp();
    } else if (window.runtime?.Quit) {
      window.runtime.Quit();
    }
  }
}
