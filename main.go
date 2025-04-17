package main

import (
	"fyne.io/systray"
	"github.com/DaniilSokolyuk/sing-vnet/internal"
)

func main() {
	systray.Run(internal.Run, func() {})
}
