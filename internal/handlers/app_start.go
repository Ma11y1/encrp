package handlers

import (
	"context"
	"encrp/internal/config"
	"encrp/internal/errors"
	"encrp/internal/logger"
	"encrp/internal/services"
	"fmt"
	"os"
)

type AppStartHandler struct {
	config   *config.Config
	handlers *Container
	services *services.Container
}

func NewAppStartHandler(cfg *config.Config, handlers *Container, services *services.Container) *AppStartHandler {
	return &AppStartHandler{config: cfg, handlers: handlers, services: services}
}

func (h *AppStartHandler) Start(ctx context.Context) error {
	fmt.Printf("v.%s\n", h.config.General.Version())

	if ctx.Err() != nil {
		return errors.WrapLog(ctx.Err(), "App.Start()", "Preliminary completion of execution")
	}

	password, pathStorage := getArg("/p"), getArg("/s")

	if password == "" {
		if err := promptInput("pwd > ", &password, 3); err != nil {
			return errors.Wrap(err, "AppStartHandler.Start()", "Error reading password")
		}
	}

	if ctx.Err() != nil {
		return errors.WrapLog(ctx.Err(), "App.Start()", "Preliminary completion of execution")
	}

	if pathStorage == "" {
		if err := promptInput("storage > ", &pathStorage, 3); err != nil {
			return errors.WrapLog(err, "AppStartHandler.Start()", "Error reading path storage")
		}
	}

	h.config.General.SetPassword(password)
	h.config.Storage.SetPath(pathStorage)

	err := h.services.Storage.LoadStorage(pathStorage)
	if err != nil {
		fmt.Printf("Error loading storage by path '%s'\n", pathStorage)
		return errors.Wrapf(err, "AppStartHandler.Start()", "Error loading storage by path '%s': %v", pathStorage, err)
	}

	if ctx.Err() != nil {
		return errors.WrapLog(ctx.Err(), "App.Start()", "Preliminary completion of execution")
	}

	return nil
}

func getArg(flag string) string {
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == flag {
			return os.Args[i+1]
		}
	}
	return ""
}

func promptInput(prompt string, input *string, attempts int) error {
	var err error
	for i := 1; i <= attempts; i++ {
		fmt.Print(prompt)
		_, err = fmt.Scanln(input)
		if *input != "" && err == nil {
			return nil
		}
		msg := fmt.Sprintf("Attempt %d, error: %v\n", i, err)
		fmt.Println(msg)
		logger.Warn("AppStartHandler.promptInput()", msg)
	}
	return err
}
