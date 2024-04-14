package router

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/transport/web/controller"
	"net/http"
)

func MakeRouter(ctx context.Context) (http.Handler, error) {
	r := mux.NewRouter()
	r.Handle("/rss", controller.RSS{})

	return r, nil
}
