package internal

import (
	"log/slog"
	"time"

	"github.com/DaniilSokolyuk/sing-vnet/ut/shell"
)

func (a *App) StartSingBox() (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.Process != nil {
		slog.Warn("SingBox is already running")
		return nil
	}

	slog.Info("Starting SingBox...")
	a.Process = shell.Exec(a.Exec, "run", "-c", a.Cfg.Sing.FileConfig).Attach()
	err = a.Process.Start()
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	cfg := Config{
		FromInterface: InterfaceConfig{
			Name:    "en0",
			Network: "172.26.0.0/16",
			LocalIP: "172.26.0.1",
		},
		ToInterface: InterfaceConfig{
			Name:    "utun128",
			Network: "172.26.0.0/16",
			LocalIP: "172.26.0.1",
		},
	}

	a.Bridge, err = Start(a.Ctx, cfg)
	if err != nil {
		slog.Error("Failed to start bridge", "error", err)
		return
	}

	return nil
}

func (a *App) StopSingBox() {
	if a.Process == nil {
		slog.Warn("SingBox is not running")
		return
	}

	a.Bridge.Close()
	a.Bridge = nil

	err := a.Process.Stop()
	if err != nil {
		slog.Error("Failed to stop SingBox", "error", err)
	}

	a.Process = nil
}
