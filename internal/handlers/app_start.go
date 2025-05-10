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

type ApplicationStartHandler struct {
	config   *config.Config
	handlers *Container
	services *services.Container
}

func NewAppStartHandler(cfg *config.Config, handlers *Container, services *services.Container) *ApplicationStartHandler {
	return &ApplicationStartHandler{
		config:   cfg,
		handlers: handlers,
		services: services,
	}
}

func (h *ApplicationStartHandler) Start(ctx context.Context) error {
	fmt.Printf("v.%s\n", h.config.General.Version())

	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "ApplicationStartHandler.Start()", "Preliminary completion of execution")
	}

	passphrase, pathStorage := getArg("/p"), getArg("/s")

	if passphrase == "" {
		if err := promptInput("pwd > ", &passphrase, 3); err != nil {
			return errors.Wrap(err, "ApplicationStartHandler.Start()", "Error reading passphrase")
		}
		if passphrase == "exit" {
			return errors.New("ApplicationStartHandler.Start()", "Premature termination")
		}
	}

	if ctx.Err() != nil {
		return errors.WrapLog(ctx.Err(), "ApplicationStartHandler.Start()", "Preliminary completion of execution")
	}

	if pathStorage == "" {
		if err := promptInput("storage > ", &pathStorage, 3); err != nil {
			return errors.Wrap(err, "ApplicationStartHandler.Start()", "Error reading path storage")
		}
		if pathStorage == "exit" {
			return errors.New("ApplicationStartHandler.Start()", "Premature termination")
		}
	}

	h.config.General.SetPassphrase(passphrase)
	h.config.Storage.SetPath(pathStorage)

	err := h.services.Storage.LoadStorage(pathStorage)
	if err != nil {
		fmt.Printf("Error loading storage by path '%s'\n", pathStorage)
		return errors.Wrapf(err, "ApplicationStartHandler.Start()", "Error loading storage by path '%s': %v", pathStorage, err)
	}

	logger.Infof("ApplicationStartHandler.Start()", "Successfully start app with storage by path '%s'", pathStorage)

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
		logger.Warn("ApplicationStartHandler.promptInput()", msg)
	}
	return err
}
