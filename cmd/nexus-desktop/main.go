package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kivervinicius/ai-cli/internal/app"
	"github.com/kivervinicius/ai-cli/internal/control/web"
	"github.com/kivervinicius/ai-cli/internal/desktop"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func main() {
	// 1. Initialize Nexus Core with ephemeral loopback port for Desktop
	core, err := app.NewCore(app.CoreConfig{
		Host:   "127.0.0.1",
		Port:   0, // Ephemeral port
		NoOpen: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Nexus Core: %v\n", err)
		os.Exit(1)
	}

	coreCtx, coreCancel := context.WithCancel(context.Background())
	defer coreCancel()

	go func() {
		if err := core.Start(coreCtx); err != nil {
			fmt.Fprintf(os.Stderr, "Nexus Core exited: %v\n", err)
		}
	}()

	select {
	case <-core.Ready():
	case <-time.After(10 * time.Second):
		fmt.Fprintln(os.Stderr, "Timeout waiting for Nexus Core to be ready")
		os.Exit(1)
	}

	// 2. Obtain identical embedded frontend assets from the core web package
	distFS, err := web.EmbeddedDistFS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to access embedded web assets: %v\n", err)
		os.Exit(1)
	}

	// 3. Provision pre-authenticated desktop session
	desktopSess, err := core.CreateDesktopSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to provision desktop session: %v\n", err)
		os.Exit(1)
	}

	// 4. Initialize native desktop bridge and window state manager
	windowManager := desktop.NewWindowStateManager("")
	windowState := windowManager.Load()
	desktopApp := desktop.NewApp(nil, windowManager, desktop.BootstrapInfo{
		ServerURL:    core.URL(),
		SessionToken: desktopSess.ID,
		CSRFToken:    desktopSess.CSRFToken,
	})

	// 5. Launch Wails native shell
	err = wails.Run(&options.App{
		Title:     "IAPro Nexus",
		Width:     windowState.Width,
		Height:    windowState.Height,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:  distFS,
			Handler: core.Handler(),
		},
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 23, A: 255},
		OnStartup:        desktopApp.Startup,
		OnShutdown: func(ctx context.Context) {
			desktopApp.Shutdown(ctx)
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = core.Stop(shutdownCtx)
		},
		Bind: []any{
			desktopApp,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "IAPro Nexus",
				Message: "Autonomous AI Orchestrator & Workspace OS",
			},
		},
		Linux: &linux.Options{
			ProgramName: "nexus-desktop",
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Wails application error: %v\n", err)
		os.Exit(1)
	}
}
