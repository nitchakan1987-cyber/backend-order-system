package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, statusCode int, body Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(body)
}

func Success(w http.ResponseWriter, message string, data any) {
	JSON(w, http.StatusOK, Envelope{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, statusCode int, code, message string) {
	JSON(w, statusCode, Envelope{
		Status:  "error",
		Message: message,
		Code:    code,
	})
}
