package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (a *app) handleDeviceTypes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.services.deviceTypes.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list device types: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	case http.MethodPost:
		var payload deviceType
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		item, err := a.services.deviceTypes.Create(payload)
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

func (a *app) handleDeviceTypeByID(w http.ResponseWriter, r *http.Request) {
	idText := strings.TrimPrefix(r.URL.Path, "/api/device-types/")
	id, err := strconv.Atoi(idText)
	if err != nil {
		http.Error(w, "invalid device type id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var payload deviceType
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		updated, ok, err := a.services.deviceTypes.Update(id, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, "device type not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updated)
	case http.MethodDelete:
		ok, err := a.services.deviceTypes.Delete(id)
		if err != nil {
			if strings.Contains(err.Error(), "in use") {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, fmt.Sprintf("failed to delete device type: %v", err), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "device type not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
