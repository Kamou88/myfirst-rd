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
	pathText := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/devices/"), "/")
	parts := strings.Split(pathText, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		http.Error(w, "invalid device id", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid device id", http.StatusBadRequest)
		return
	}
	if len(parts) == 2 && parts[1] == "unlock" {
		a.handleDeviceUnlock(w, r, id)
		return
	}
	if len(parts) != 1 {
		http.Error(w, "invalid device path", http.StatusBadRequest)
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

func (a *app) handleDeviceUnlock(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload deviceUnlockPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	ok, err := a.services.devices.UpdateUnlock(id, payload.IsUnlocked)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
