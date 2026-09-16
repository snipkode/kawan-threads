package handler

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the canonical envelope for every JSON response.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message *string     `json:"message,omitempty"`
	Code    *string     `json:"code,omitempty"`
}

// WriteJSON marshals data as JSON and writes it with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteSuccess writes a 200 OK success response wrapping data.
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// WriteCreated writes a 201 Created success response wrapping data.
func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error response with the given HTTP status, a human-
// readable message, and an optional machine-readable error code.
func WriteError(w http.ResponseWriter, status int, message, code string) {
	msg := message
	c := code
	WriteJSON(w, status, APIResponse{
		Success: false,
		Data:    nil,
		Message: &msg,
		Code:    &c,
	})
}

// WriteNotFound writes a 404 Not Found error response.
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, message, "NOT_FOUND")
}

// WriteBadRequest writes a 400 Bad Request error response.
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, message, "BAD_REQUEST")
}

// WriteInternalError writes a 500 Internal Server Error response.
func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, message, "INTERNAL_ERROR")
}
