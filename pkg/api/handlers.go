package api

import (
	"encoding/json"
	"net/http"
	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) StartJob(w http.ResponseWriter, r *http.Request) {
	var req StartJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_request", "message": err.Error()})
		return
	}
	resp, code, err := h.Core.TryStartJob(req)
	if err != nil {
		writeJSON(w, code, map[string]string{"error": "start_failed", "message": err.Error()})
		return
	}
	writeJSON(w, code, resp)
}

func (h *Handlers) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "job_id")
	resp, code, err := h.Core.GetJob(id)
	if err != nil {
		writeJSON(w, code, map[string]string{"error": "not_found", "message": err.Error()})
		return
	}
	writeJSON(w, code, resp)
}

func (h *Handlers) StopJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "job_id")
	resp, code, err := h.Core.StopJob(id)
	if err != nil {
		writeJSON(w, code, map[string]string{"error": "stop_failed", "message": err.Error()})
		return
	}
	writeJSON(w, code, resp)
}

func (h *Handlers) GetResults(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "job_id")
	resp, code, err := h.Core.GetResults(id)
	if err != nil {
		writeJSON(w, code, map[string]string{"error": "results_failed", "message": err.Error()})
		return
	}
	writeJSON(w, code, resp)
}
