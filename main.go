package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

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
	application := &app{
		services: appServices{
			recipes: recipeService{
				repo: recipeRepository{db: db},
			},
			devices: deviceService{
				repo:     deviceRepository{db: db},
				typeRepo: deviceTypeRepository{db: db},
			},
			materials: materialService{
				repo: materialRepository{db: db},
			},
			deviceTypes: deviceTypeService{
				repo: deviceTypeRepository{db: db},
			},
		},
	}
	application.registerRoutes(mux)

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
