package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorEnvelope struct {
	Error struct {
		Code          string         `json:"code"`
		Message       string         `json:"message"`
		CorrelationID string         `json:"correlation_id"`
		Details       map[string]any `json:"details,omitempty"`
	} `json:"error"`
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string, details map[string]any) {
	corrID := r.Header.Get("X-Correlation-Id")
	if corrID == "" {
		corrID = "corr-system-gen"
	}

	env := ErrorEnvelope{}
	env.Error.Code = code
	env.Error.Message = message
	env.Error.CorrelationID = corrID
	env.Error.Details = details

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
