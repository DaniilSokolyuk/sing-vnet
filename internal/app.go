package internal

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"fyne.io/systray"
	"github.com/DaniilSokolyuk/sing-vnet/ut"
	"github.com/DaniilSokolyuk/sing-vnet/ut/shell"
	"github.com/DaniilSokolyuk/sing-vnet/webui"
)

const (
	UIPort = ":8399"
)

var app *App

type App struct {
	Cfg     Conf
	Exec    string
	Process *shell.Shell
	Bridge  *Bridge
	Ctx     context.Context
	Stop    context.CancelFunc

	mu sync.Mutex
}

func Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	app = &App{
		Cfg:  LoadConfig(),
		Ctx:  ctx,
		Stop: stop,
	}

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	if !ut.CheckRoot() {
		os.Exit(1)
	}

	go TrayOnReady()

	go func() {
		if err := webui.StartServer(UIPort); err != nil {
			log.Fatal(err)
		}
	}()

	//browser.OpenURL("http://127.0.0.1" + UIPort)

	slog.Info("Downloading SingBox...")
	app.Exec = ut.TryOrPanic(app.DownloadLatestSingBoxIfNotExists)
	fmt.Println("SingBox singExec:", app.Exec)

	<-ctx.Done()

	app.StopSingBox()
	systray.Quit()
}
