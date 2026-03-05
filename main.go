package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

type healthResponse struct {
	Message string `json:"message"`
}

type recipeMaterial struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type recipe struct {
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	MachineName  string           `json:"machineName"`
	CycleSeconds float64          `json:"cycleSeconds"`
	PowerKW      float64          `json:"powerKW"`
	Inputs       []recipeMaterial `json:"inputs"`
	Outputs      []recipeMaterial `json:"outputs"`
}

type device struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	EfficiencyPercent float64 `json:"efficiencyPercent"`
}

type material struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:5173"
	}

	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = "recipes.db"
	}
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(".", dbPath)
	}

	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("failed to open sqlite db: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{
			Message: "go backend is running",
		})
	})
	mux.HandleFunc("/api/recipes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := listRecipes(db)
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

			if err := validateRecipe(payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			item, err := createRecipe(db, recipe{
				Name:         strings.TrimSpace(payload.Name),
				MachineName:  strings.TrimSpace(payload.MachineName),
				CycleSeconds: payload.CycleSeconds,
				PowerKW:      payload.PowerKW,
				Inputs:       sanitizeMaterials(payload.Inputs),
				Outputs:      sanitizeMaterials(payload.Outputs),
			})
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to save recipe: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(item)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/devices", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := listDevices(db)
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
			if err := validateDevice(payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			item, err := createDevice(db, device{
				Name:              strings.TrimSpace(payload.Name),
				EfficiencyPercent: payload.EfficiencyPercent,
			})
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to save device: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(item)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/devices/", func(w http.ResponseWriter, r *http.Request) {
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
			payload.ID = id
			if err := validateDevice(payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			updated, ok, err := updateDevice(db, payload)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to update device: %v", err), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, "device not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(updated)
		case http.MethodDelete:
			ok, err := deleteDevice(db, id)
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
	})
	mux.HandleFunc("/api/materials", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := listMaterials(db)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to list materials: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(items)
		case http.MethodPost:
			var payload material
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
			if err := validateMaterial(payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			item, err := createMaterial(db, material{Name: strings.TrimSpace(payload.Name)})
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to save material: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(item)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/materials/", func(w http.ResponseWriter, r *http.Request) {
		idText := strings.TrimPrefix(r.URL.Path, "/api/materials/")
		id, err := strconv.Atoi(idText)
		if err != nil {
			http.Error(w, "invalid material id", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPut:
			var payload material
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
			payload.ID = id
			if err := validateMaterial(payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			updated, ok, err := updateMaterial(db, payload)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to update material: %v", err), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, "material not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(updated)
		case http.MethodDelete:
			ok, err := deleteMaterial(db, id)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to delete material: %v", err), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, "material not found", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: withCORS(frontendOrigin, mux),
	}

	log.Printf("go backend listening on http://localhost:%s", port)
	log.Printf("sqlite db path: %s", dbPath)
	log.Fatal(server.ListenAndServe())
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func initSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS recipes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  machine_name TEXT NOT NULL,
  cycle_seconds REAL NOT NULL,
  power_kw REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS recipe_materials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  recipe_id INTEGER NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('input','output')),
  name TEXT NOT NULL,
  amount REAL NOT NULL,
  position INTEGER NOT NULL,
  FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS devices (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  efficiency_percent REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS materials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);
`
	_, err := db.Exec(schema)
	return err
}

func createRecipe(db *sql.DB, item recipe) (recipe, error) {
	tx, err := db.Begin()
	if err != nil {
		return recipe{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.Exec(
		`INSERT INTO recipes (name, machine_name, cycle_seconds, power_kw) VALUES (?, ?, ?, ?)`,
		item.Name, item.MachineName, item.CycleSeconds, item.PowerKW,
	)
	if err != nil {
		return recipe{}, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return recipe{}, err
	}
	item.ID = int(id64)

	for i, m := range item.Inputs {
		_, err = tx.Exec(
			`INSERT INTO recipe_materials (recipe_id, kind, name, amount, position) VALUES (?, 'input', ?, ?, ?)`,
			item.ID, m.Name, m.Amount, i,
		)
		if err != nil {
			return recipe{}, err
		}
	}
	for i, m := range item.Outputs {
		_, err = tx.Exec(
			`INSERT INTO recipe_materials (recipe_id, kind, name, amount, position) VALUES (?, 'output', ?, ?, ?)`,
			item.ID, m.Name, m.Amount, i,
		)
		if err != nil {
			return recipe{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		return recipe{}, err
	}
	return item, nil
}

func listRecipes(db *sql.DB) ([]recipe, error) {
	rows, err := db.Query(`
SELECT
  r.id,
  r.name,
  r.machine_name,
  r.cycle_seconds,
  r.power_kw,
  m.kind,
  m.name,
  m.amount
FROM recipes r
LEFT JOIN recipe_materials m ON m.recipe_id = r.id
ORDER BY r.id ASC, m.kind ASC, m.position ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int]*recipe)
	order := make([]int, 0)
	for rows.Next() {
		var (
			id           int
			name         string
			machineName  string
			cycleSeconds float64
			powerKW      float64
			kind         sql.NullString
			materialName sql.NullString
			amount       sql.NullFloat64
		)
		if err := rows.Scan(&id, &name, &machineName, &cycleSeconds, &powerKW, &kind, &materialName, &amount); err != nil {
			return nil, err
		}

		item, ok := byID[id]
		if !ok {
			item = &recipe{
				ID:           id,
				Name:         name,
				MachineName:  machineName,
				CycleSeconds: cycleSeconds,
				PowerKW:      powerKW,
				Inputs:       make([]recipeMaterial, 0),
				Outputs:      make([]recipeMaterial, 0),
			}
			byID[id] = item
			order = append(order, id)
		}

		if kind.Valid && materialName.Valid && amount.Valid {
			mat := recipeMaterial{Name: materialName.String, Amount: amount.Float64}
			if kind.String == "input" {
				item.Inputs = append(item.Inputs, mat)
			}
			if kind.String == "output" {
				item.Outputs = append(item.Outputs, mat)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]recipe, 0, len(order))
	for _, id := range order {
		result = append(result, *byID[id])
	}
	return result, nil
}

func createDevice(db *sql.DB, item device) (device, error) {
	res, err := db.Exec(
		`INSERT INTO devices (name, efficiency_percent) VALUES (?, ?)`,
		item.Name, item.EfficiencyPercent,
	)
	if err != nil {
		return device{}, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return device{}, err
	}
	item.ID = int(id64)
	return item, nil
}

func listDevices(db *sql.DB) ([]device, error) {
	rows, err := db.Query(`
SELECT id, name, efficiency_percent
FROM devices
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]device, 0)
	for rows.Next() {
		var item device
		if err := rows.Scan(&item.ID, &item.Name, &item.EfficiencyPercent); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func updateDevice(db *sql.DB, item device) (device, bool, error) {
	res, err := db.Exec(
		`UPDATE devices SET name = ?, efficiency_percent = ? WHERE id = ?`,
		item.Name, item.EfficiencyPercent, item.ID,
	)
	if err != nil {
		return device{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return device{}, false, err
	}
	return item, affected > 0, nil
}

func deleteDevice(db *sql.DB, id int) (bool, error) {
	res, err := db.Exec(`DELETE FROM devices WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func createMaterial(db *sql.DB, item material) (material, error) {
	res, err := db.Exec(`INSERT INTO materials (name) VALUES (?)`, item.Name)
	if err != nil {
		return material{}, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return material{}, err
	}
	item.ID = int(id64)
	return item, nil
}

func listMaterials(db *sql.DB) ([]material, error) {
	rows, err := db.Query(`
SELECT id, name
FROM materials
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]material, 0)
	for rows.Next() {
		var item material
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func updateMaterial(db *sql.DB, item material) (material, bool, error) {
	res, err := db.Exec(`UPDATE materials SET name = ? WHERE id = ?`, item.Name, item.ID)
	if err != nil {
		return material{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return material{}, false, err
	}
	return item, affected > 0, nil
}

func deleteMaterial(db *sql.DB, id int) (bool, error) {
	res, err := db.Exec(`DELETE FROM materials WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func withCORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func validateRecipe(item recipe) error {
	if strings.TrimSpace(item.Name) == "" {
		return errText("recipe name is required")
	}
	if strings.TrimSpace(item.MachineName) == "" {
		return errText("machine name is required")
	}
	if item.CycleSeconds <= 0 {
		return errText("cycleSeconds must be greater than 0")
	}
	if item.PowerKW < 0 {
		return errText("powerKW cannot be negative")
	}
	if len(item.Inputs) == 0 {
		return errText("at least one input material is required")
	}
	if len(item.Outputs) == 0 {
		return errText("at least one output material is required")
	}
	for _, m := range item.Inputs {
		if strings.TrimSpace(m.Name) == "" {
			return errText("input material name is required")
		}
		if m.Amount <= 0 {
			return errText("input material amount must be greater than 0")
		}
	}
	for _, m := range item.Outputs {
		if strings.TrimSpace(m.Name) == "" {
			return errText("output material name is required")
		}
		if m.Amount <= 0 {
			return errText("output material amount must be greater than 0")
		}
	}
	return nil
}

func validateDevice(item device) error {
	if strings.TrimSpace(item.Name) == "" {
		return errText("device name is required")
	}
	if item.EfficiencyPercent <= 0 {
		return errText("efficiencyPercent must be greater than 0")
	}
	return nil
}

func validateMaterial(item material) error {
	if strings.TrimSpace(item.Name) == "" {
		return errText("material name is required")
	}
	return nil
}

func sanitizeMaterials(items []recipeMaterial) []recipeMaterial {
	out := make([]recipeMaterial, 0, len(items))
	for _, item := range items {
		out = append(out, recipeMaterial{
			Name:   strings.TrimSpace(item.Name),
			Amount: item.Amount,
		})
	}
	return out
}

type errText string

func (e errText) Error() string {
	return string(e)
}
