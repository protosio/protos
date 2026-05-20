//go:build cgo

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/getlantern/systray"
	"github.com/protosio/protos/cmd/protosc/icon"
)

func runClientApp() error {
	systray.Run(onReady, onExit)
	return nil
}

func onReady() {
	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTooltip("Protos")
	mQuit := systray.AddMenuItem("Quit", "Quit")

	osSigs := make(chan os.Signal, 1)
	errorSig := make(chan struct{}, 1)
	signal.Notify(osSigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer signal.Stop(osSigs)
		handleQuitSignals(osSigs, mQuit.ClickedCh, errorSig, systray.Quit)
	}()

	if err := startClient(); err != nil {
		log.Error(err)
		errorSig <- struct{}{}
	}
}

func onExit() {
	log.Info("shutdown complete")
}
