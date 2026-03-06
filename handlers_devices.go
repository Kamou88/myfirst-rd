package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (a *app) handleDevices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.services.devices.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list devices: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	case http.MethodPost:
		var payload device
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		item, err := a.services.devices.Create(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(item)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) handleDeviceByID(w http.ResponseWriter, r *http.Request) {
	idText := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	id, err := strconv.Atoi(idText)
	if err != nil {
		http.Error(w, "invalid device id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var payload device
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		updated, ok, err := a.services.devices.Update(id, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updated)
	case http.MethodDelete:
		ok, err := a.services.devices.Delete(id)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to delete device: %v", err), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
