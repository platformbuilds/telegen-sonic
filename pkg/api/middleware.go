package api

import (
	"log"
	"net/http"
	"time"
)

type Handlers struct {
	Core Core
}

type Core interface {
	TryStartJob(StartJobRequest) (StartJobResponse, int, error)
	GetJob(id string) (JobStatus, int, error)
	StopJob(id string) (StopJobResponse, int, error)
	GetResults(id string) (JobResults, int, error)
}

func (h *Handlers) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
