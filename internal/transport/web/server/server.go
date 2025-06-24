package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

type Server struct {
	TLSDisabled       bool
	TLSDisabledPort   int
	AutocertHostnames []string
	Router            http.Handler
}

func (s *Server) Run(ctx context.Context) error {
	server := &http.Server{
		Handler:      s.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if s.TLSDisabled {
		server.Addr = fmt.Sprintf(":%d", s.TLSDisabledPort)
		return server.ListenAndServe()
	} else {
		return server.Serve(autocert.NewListener(s.AutocertHostnames...))
	}
}
