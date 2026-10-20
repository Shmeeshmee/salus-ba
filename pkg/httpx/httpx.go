// Package httpx holds small helpers for JSON HTTP handlers.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status is already sent; a failed write means the client went away.
	_ = json.NewEncoder(w).Encode(v)
}

// InternalError logs err and sends a fixed 500 response. The error text never reaches the client.
func InternalError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	log.ErrorContext(r.Context(), "internal error", "method", r.Method, "route", r.Pattern, "error", err)
	JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}
