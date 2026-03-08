package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrateSchema(db); err != nil {
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
  device_model TEXT NOT NULL DEFAULT '',
  device_id INTEGER,
  cycle_seconds REAL NOT NULL,
  power_kw REAL NOT NULL,
  can_speedup INTEGER NOT NULL DEFAULT 1,
  can_boost INTEGER NOT NULL DEFAULT 1,
  is_researched INTEGER NOT NULL DEFAULT 0,
  effect_mode TEXT NOT NULL DEFAULT 'speed',
  booster_tier TEXT NOT NULL DEFAULT 'mk3',
  FOREIGN KEY(device_id) REFERENCES devices(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS recipe_materials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  recipe_id INTEGER NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('input','output')),
  material_id INTEGER,
  name TEXT NOT NULL,
  amount REAL NOT NULL,
  position INTEGER NOT NULL,
  FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE,
  FOREIGN KEY(material_id) REFERENCES materials(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS devices (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  device_type TEXT NOT NULL DEFAULT '',
  efficiency_percent REAL NOT NULL,
  power_kw REAL NOT NULL DEFAULT 0,
  is_unlocked INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS materials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  is_craftable INTEGER NOT NULL DEFAULT 0,
  is_raw INTEGER NOT NULL DEFAULT 0,
  rarity TEXT NOT NULL DEFAULT '一般'
);

CREATE TABLE IF NOT EXISTS device_types (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS production_lines (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS production_line_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  line_id INTEGER NOT NULL,
  recipe_id INTEGER NOT NULL,
  machine_count INTEGER NOT NULL,
  position INTEGER NOT NULL,
  FOREIGN KEY(line_id) REFERENCES production_lines(id) ON DELETE CASCADE,
  FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
);
`
	_, err := db.Exec(schema)
	return err
}

func migrateSchema(db *sql.DB) error {
	hasPowerKW, err := tableHasColumn(db, "devices", "power_kw")
	if err != nil {
		return err
	}
	if !hasPowerKW {
		if _, err := db.Exec(`ALTER TABLE devices ADD COLUMN power_kw REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	hasDeviceType, err := tableHasColumn(db, "devices", "device_type")
	if err != nil {
		return err
	}
	if !hasDeviceType {
		if _, err := db.Exec(`ALTER TABLE devices ADD COLUMN device_type TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}

	hasDeviceUnlocked, err := tableHasColumn(db, "devices", "is_unlocked")
	if err != nil {
		return err
	}
	if !hasDeviceUnlocked {
		if _, err := db.Exec(`ALTER TABLE devices ADD COLUMN is_unlocked INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`UPDATE devices SET device_type = '未分类' WHERE TRIM(device_type) = ''`); err != nil {
		return err
	}

	hasRecipeDeviceModel, err := tableHasColumn(db, "recipes", "device_model")
	if err != nil {
		return err
	}

	hasRecipeDeviceID, err := tableHasColumn(db, "recipes", "device_id")
	if err != nil {
		return err
	}
	if !hasRecipeDeviceID {
		if _, err := db.Exec(`ALTER TABLE recipes ADD COLUMN device_id INTEGER`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
UPDATE recipes
SET device_id = (
  SELECT d.id
  FROM devices d
  WHERE d.name = recipes.device_model
    AND d.device_type = recipes.machine_name
  ORDER BY d.id ASC
  LIMIT 1
)
WHERE device_id IS NULL
`); err != nil {
		return err
	}
	if !hasRecipeDeviceModel {
		if _, err := db.Exec(`ALTER TABLE recipes ADD COLUMN device_model TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}

	hasCanSpeedup, err := tableHasColumn(db, "recipes", "can_speedup")
	if err != nil {
		return err
	}
	if !hasCanSpeedup {
		if _, err := db.Exec(`ALTER TABLE recipes ADD COLUMN can_speedup INTEGER NOT NULL DEFAULT 1`); err != nil {
			return err
		}
	}

	hasCanBoost, err := tableHasColumn(db, "recipes", "can_boost")
	if err != nil {
		return err
	}
	if !hasCanBoost {
		if _, err := db.Exec(`ALTER TABLE recipes ADD COLUMN can_boost INTEGER NOT NULL DEFAULT 1`); err != nil {
			return err
		}
	}

	hasIsResearched, err := tableHasColumn(db, "recipes", "is_researched")
	if err != nil {
		return err
	}
	if !hasIsResearched {
		if _, err := db.Exec(`ALTER TABLE recipes ADD COLUMN is_researched INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	hasEffectMode, err := tableHasColumn(db, "recipes", "effect_mode")
	if err != nil {
		return err
	}
	if !hasEffectMode {
		if _, err := db.Exec(`ALTER TABLE recipes ADD COLUMN effect_mode TEXT NOT NULL DEFAULT 'speed'`); err != nil {
			return err
		}
	}

	hasBoosterTier, err := tableHasColumn(db, "recipes", "booster_tier")
	if err != nil {
		return err
	}
	if !hasBoosterTier {
		if _, err := db.Exec(`ALTER TABLE recipes ADD COLUMN booster_tier TEXT NOT NULL DEFAULT 'mk3'`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`UPDATE recipes SET booster_tier = 'mk3' WHERE TRIM(booster_tier) = ''`); err != nil {
		return err
	}

	if _, err := db.Exec(`
INSERT OR IGNORE INTO device_types(name)
SELECT DISTINCT device_type FROM devices WHERE TRIM(device_type) <> ''
`); err != nil {
		return err
	}

	hasMaterialCraftable, err := tableHasColumn(db, "materials", "is_craftable")
	if err != nil {
		return err
	}
	if !hasMaterialCraftable {
		if _, err := db.Exec(`ALTER TABLE materials ADD COLUMN is_craftable INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	hasMaterialRarity, err := tableHasColumn(db, "materials", "rarity")
	if err != nil {
		return err
	}
	if !hasMaterialRarity {
		if _, err := db.Exec(`ALTER TABLE materials ADD COLUMN rarity TEXT NOT NULL DEFAULT '一般'`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`UPDATE materials SET rarity = '一般' WHERE TRIM(rarity) = ''`); err != nil {
		return err
	}

	hasMaterialIsRaw, err := tableHasColumn(db, "materials", "is_raw")
	if err != nil {
		return err
	}
	if !hasMaterialIsRaw {
		if _, err := db.Exec(`ALTER TABLE materials ADD COLUMN is_raw INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	hasRecipeMaterialID, err := tableHasColumn(db, "recipe_materials", "material_id")
	if err != nil {
		return err
	}
	if !hasRecipeMaterialID {
		if _, err := db.Exec(`ALTER TABLE recipe_materials ADD COLUMN material_id INTEGER`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
UPDATE recipe_materials
SET material_id = (
  SELECT m.id
  FROM materials m
  WHERE m.name = recipe_materials.name
  ORDER BY m.id ASC
  LIMIT 1
)
WHERE material_id IS NULL
`); err != nil {
		return err
	}
	if err := syncMaterialRawByRecipeInputs(db); err != nil {
		return err
	}
	if err := backfillRecipeBoosterVariants(db); err != nil {
		return err
	}
	return nil
}

func syncMaterialRawByRecipeInputs(db *sql.DB) error {
	if _, err := db.Exec(`UPDATE materials SET is_raw = 0`); err != nil {
		return err
	}
	if _, err := db.Exec(`
UPDATE materials
SET is_raw = 1
WHERE id IN (
  SELECT DISTINCT rm.material_id
  FROM recipe_materials rm
  WHERE rm.kind = 'input' AND rm.material_id IS NOT NULL
)
`); err != nil {
		return err
	}
	if _, err := db.Exec(`
UPDATE materials
SET is_raw = 1
WHERE name IN (
  SELECT DISTINCT TRIM(rm.name)
  FROM recipe_materials rm
  WHERE rm.kind = 'input'
    AND (rm.material_id IS NULL OR rm.material_id = 0)
    AND TRIM(rm.name) <> ''
)
`); err != nil {
		return err
	}
	return nil
}

func tableHasColumn(db *sql.DB, tableName string, columnName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
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
