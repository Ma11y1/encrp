package app

import (
	"context"
	"encrp/internal/config"
	"encrp/internal/errors"
	"encrp/internal/handlers"
	"encrp/internal/services"
)

type App struct {
	config    *config.Config
	services  *services.Container
	handlers  *handlers.Container
	cancel    context.CancelFunc
	isStarted bool
}

func NewApp() (*App, error) {
	a := &App{
		config:   config.NewConfig(),
		services: &services.Container{},
		handlers: &handlers.Container{},
	}

	a.services.Crypt = services.NewCryptAESGCMService(a.config, a.services)
	a.services.Keys = services.NewKeyService(a.config, a.services)
	a.services.Storage = services.NewFileStorageService(a.config, a.services)

	a.handlers.AppStart = handlers.NewAppStartHandler(a.config, a.handlers, a.services)
	a.handlers.AppStop = handlers.NewAppStopHandler(a.config, a.handlers, a.services)
	a.handlers.CommandProcessor = handlers.NewCommandProcessorHandler(a.config, a.handlers, a.services)

	return a, nil
}

func (a *App) Start(ctx context.Context) error {
	if a.isStarted {
		return errors.New("App.Start()", "App is already started")
	}
	defer func() { a.Stop() }()
	a.isStarted = true

	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	err := a.handlers.AppStart.Start(ctx)
	if err != nil {
		return errors.WrapLog(err, "App.Start()", "Error starting app")
	}

	err = a.handlers.CommandProcessor.Start(ctx)
	if err != nil {
		return errors.WrapLog(err, "App.Start()", "Error command processing")
	}

	return nil
}

func (a *App) Stop() error {
	if !a.isStarted {
		return errors.New("App.Stop()", "App is already stopped")
	}

	a.isStarted = false
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}

	err := a.handlers.AppStop.Start(context.Background())
	if err != nil {
		return errors.WrapLog(err, "App.Stop()", "Error stopping app")
	}

	return nil
}

func (a *App) IsStarted() bool {
	return a.isStarted
}
