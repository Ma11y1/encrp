package handlers

import (
	"context"
	"encrp/internal/config"
	"encrp/internal/services"
)

type ApplicationStopHandler struct {
	config   *config.Config
	handlers *Container
	services *services.Container
}

func NewAppStopHandler(cfg *config.Config, handlers *Container, services *services.Container) *ApplicationStopHandler {
	return &ApplicationStopHandler{
		config:   cfg,
		handlers: handlers,
		services: services,
	}
}

func (a *ApplicationStopHandler) Start(context.Context) error {
	a.config.General.WipePassphrase()
	a.services.Storage.WipeStorage()
	return nil
}
