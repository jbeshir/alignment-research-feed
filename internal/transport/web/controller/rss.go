package controller

import "net/http"

type RSS struct{}

func (c RSS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
}
