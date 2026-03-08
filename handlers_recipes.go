package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (a *app) handleRecipes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.services.recipes.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list recipes: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	case http.MethodPost:
		var payload recipe
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		items, err := a.services.recipes.Create(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(items)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) handleRecipeByID(w http.ResponseWriter, r *http.Request) {
	pathText := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/recipes/"), "/")
	parts := strings.Split(pathText, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	if len(parts) == 2 && parts[1] == "booster" {
		a.handleRecipeBooster(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "research" {
		a.handleRecipeResearch(w, r, id)
		return
	}
	if len(parts) != 1 {
		http.Error(w, "invalid recipe path", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var payload recipe
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		updated, ok, err := a.services.recipes.ReplaceGroup(id, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, "recipe not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updated)
	case http.MethodDelete:
		ok, err := a.services.recipes.Delete(id)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to delete recipe: %v", err), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "recipe not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) handleRecipeBooster(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload recipeBoosterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	updated, ok, err := a.services.recipes.UpdateBooster(id, payload.BoosterTier)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, "recipe not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (a *app) handleRecipeResearch(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload recipeResearchPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	ok, err := a.services.recipes.UpdateResearch(id, payload.IsResearched)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, "recipe not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
