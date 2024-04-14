package app

import (
	"context"
	"fmt"
	"github.com/jbeshir/alignment-research-feed/internal/transport/web/router"
	"github.com/jbeshir/alignment-research-feed/internal/transport/web/server"
)

type Component interface {
	Run(ctx context.Context) error
}

func Setup(ctx context.Context) ([]Component, error) {
	httpRouter, err := router.MakeRouter(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create HTTP router: %w", err)
	}

	return []Component{
		&server.Server{
			TLSDisabled:      MustGetEnvAsBoolean(ctx, "HTTP_TLS_DISABLED"),
			TLSDisabledPort:  MustGetEnvAsInt(ctx, "HTTP_TLS_DISABLED_PORT"),
			AutocertHostname: MustGetEnvAsString(ctx, "HTTP_AUTOCERT_HOSTNAME"),
			Router:           httpRouter,
		},
	}, nil
}
