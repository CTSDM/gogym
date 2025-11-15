package util

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func RespondWithError(w http.ResponseWriter, r *http.Request, code int, msg string, err error) {
	RespondWithJSON(w, r, code, ErrorResponse{
		Error: msg,
	})
}

func RespondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		requestID, _ := RequestIDFromContext(r.Context())
		slog.Error("error marshalling JSON",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
			slog.String("path", r.URL.Path),
			slog.String("method", r.Method))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		requestID, _ := RequestIDFromContext(r.Context())
		slog.Debug("could not write the response",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
			slog.String("path", r.URL.Path),
			slog.String("method", r.Method))
	}
}
