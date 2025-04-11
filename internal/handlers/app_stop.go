package handlers

import (
	"context"
	"encrp/internal/config"
	"encrp/internal/services"
)

type AppStopHandler struct {
	config   *config.Config
	handlers *Container
	services *services.Container
}

func NewAppStopHandler(cfg *config.Config, handlers *Container, services *services.Container) *AppStopHandler {
	return &AppStopHandler{
		config:   cfg,
		handlers: handlers,
		services: services,
	}
}

func (a *AppStopHandler) Start(context.Context) error {
	a.config.General.WipePassword()
	a.services.Storage.WipeStorage()
	return nil
}
