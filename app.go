package main

import "net/http"

type app struct {
	services appServices
}

func (a *app) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/recipes", a.handleRecipes)
	mux.HandleFunc("/api/recipes/", a.handleRecipeByID)
	mux.HandleFunc("/api/devices", a.handleDevices)
	mux.HandleFunc("/api/devices/", a.handleDeviceByID)
	mux.HandleFunc("/api/materials", a.handleMaterials)
	mux.HandleFunc("/api/materials/", a.handleMaterialByID)
	mux.HandleFunc("/api/device-types", a.handleDeviceTypes)
	mux.HandleFunc("/api/device-types/", a.handleDeviceTypeByID)
	mux.HandleFunc("/api/production-lines", a.handleProductionLines)
	mux.HandleFunc("/api/production-lines/", a.handleProductionLineByID)
}
