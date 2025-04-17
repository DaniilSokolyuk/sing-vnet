package internal

import (
	"log/slog"

	"fyne.io/systray"
	"fyne.io/systray/example/icon"
	"github.com/pkg/browser"
)

func TrayOnReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("sing-vnet")
	systray.SetTooltip("sing-vnet")
	toggle := systray.AddMenuItem("Start", "Start proxy")
	ui := systray.AddMenuItem("UI", "Open the UI")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	// Sets the icon of a menu item.
	mQuit.SetIcon(icon.Data)
	toggle.SetIcon(icon.Data)
	ui.SetIcon(icon.Data)

	for {
		select {
		case <-toggle.ClickedCh:
			if app.Process == nil {
				err := app.StartSingBox()
				if err != nil {
					slog.Error("Failed to start SingBox", "error", err)
					continue
				}

				toggle.SetTitle("Stop")
			} else {
				app.StopSingBox()
				toggle.SetTitle("Start")
			}
		case <-ui.ClickedCh:
			browser.OpenURL("http://127.0.0.1" + UIPort)
		case <-mQuit.ClickedCh:
			app.Stop()
		}
	}
}
