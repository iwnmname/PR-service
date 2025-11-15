package handler

import (
	"encoding/json"
	"net/http"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, status int, code domain.ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := domain.NewErrorResponse(code, message)

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		http.Error(w, "failed to encode error", http.StatusInternalServerError)
	}
}
