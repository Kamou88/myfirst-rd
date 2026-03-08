package main

import (
	"database/sql"
	"net/http"
)

type app struct {
	services appServices
	db       *sql.DB
}

func (a *app) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/auth/login", a.handleAuthLogin)
	mux.HandleFunc("/api/auth/me", a.withAuth(a.handleAuthMe))
	mux.HandleFunc("/api/auth/logout", a.withAuth(a.handleAuthLogout))
	mux.HandleFunc("/api/users", a.withAuth(a.handleUsers))
	mux.HandleFunc("/api/users/", a.withAuth(a.handleUserByID))

	mux.HandleFunc("/api/recipes", a.withAuth(a.handleRecipes))
	mux.HandleFunc("/api/recipes/", a.withAuth(a.handleRecipeByID))
	mux.HandleFunc("/api/requirements/calculate", a.withAuth(a.handleRequirementCalculate))
	mux.HandleFunc("/api/requirement-plans", a.withAuth(a.handleRequirementPlans))
	mux.HandleFunc("/api/requirement-plans/", a.withAuth(a.handleRequirementPlanByID))
	mux.HandleFunc("/api/devices", a.withAuth(a.handleDevices))
	mux.HandleFunc("/api/devices/", a.withAuth(a.handleDeviceByID))
	mux.HandleFunc("/api/materials", a.withAuth(a.handleMaterials))
	mux.HandleFunc("/api/materials/sync-raw", a.withAuth(a.handleMaterialSyncRaw))
	mux.HandleFunc("/api/materials/", a.withAuth(a.handleMaterialByID))
	mux.HandleFunc("/api/device-types", a.withAuth(a.handleDeviceTypes))
	mux.HandleFunc("/api/device-types/", a.withAuth(a.handleDeviceTypeByID))
	mux.HandleFunc("/api/production-lines", a.withAuth(a.handleProductionLines))
	mux.HandleFunc("/api/production-lines/", a.withAuth(a.handleProductionLineByID))
}
