package handlers

import (
	"encoding/json"
	"net/http"
)

type errorPayload struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
	details map[string]any,
) {
	WriteJSON(w, status, errorPayload{
		Error: errorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func WriteMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", methods[0])
		if len(methods) > 1 {
			w.Header()["Allow"] = methods
		}
	}
	WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", nil)
}
