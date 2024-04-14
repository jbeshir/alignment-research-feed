package server

import (
	"context"
	"fmt"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
)

type Server struct {
	TLSDisabled      bool
	TLSDisabledPort  int
	AutocertHostname string
	Router           http.Handler
}

func (s *Server) Run(ctx context.Context) error {
	if s.TLSDisabled {
		return http.ListenAndServe(fmt.Sprintf(":%d", s.TLSDisabledPort), s.Router)
	} else {
		return http.Serve(autocert.NewListener(s.AutocertHostname), s.Router)
	}
}
