package handlers

import "context"

type Handler interface {
	Start(ctx context.Context) error
}
