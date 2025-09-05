// Copyright (c) OpenMMLab. All rights reserved.

package httpserver

import (
	"encoding/json"
	"net/http"

	"deeptrace/pkg/agent/util/storage"

	"github.com/gorilla/mux"
)

func NewDefaultHandler(storage *storage.EventStorage) *DefaultHandler {
	return &DefaultHandler{storage: storage}
}

func (h *DefaultHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/event/webhook", h.handleWebhook).Methods("POST")
}

func (h *DefaultHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var req SendEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// no severity info right now, set default value 0
	severity := int32(0)

	_, err := h.storage.StoreEvent(storage.EventEntry{
		Source:   "trainning",
		Type:     "alert",
		Message:  req.Content.Text,
		Severity: severity,
	})
	if err != nil {
		http.Error(w, "Failed to store alert message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
