//go:build linux

package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(h *Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(h.LoggingMiddleware)
	r.Route("/v1", func(r chi.Router) {
		r.Route("/monitor/jobs", func(r chi.Router) {
			r.Post("/", h.StartJob)
			r.Route("/{job_id}", func(r chi.Router) {
				r.Get("/", h.GetJob)
				r.Delete("/", h.StopJob)
				r.Get("/results", h.GetResults)
			})
		})
	})
	return r
}
