package main

import (
	"database/sql"
	"strings"
)

func createMaterial(db *sql.DB, item material) (material, error) {
	res, err := db.Exec(`INSERT INTO materials (name, is_craftable) VALUES (?, ?)`, item.Name, boolToInt(item.IsCraftable))
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
SELECT id, name, is_craftable
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
		var isCraftable int
		if err := rows.Scan(&item.ID, &item.Name, &isCraftable); err != nil {
			return nil, err
		}
		item.IsCraftable = isCraftable != 0
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func updateMaterial(db *sql.DB, item material) (material, bool, error) {
	res, err := db.Exec(
		`UPDATE materials SET name = ?, is_craftable = ? WHERE id = ?`,
		item.Name, boolToInt(item.IsCraftable), item.ID,
	)
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

func validateMaterial(item material) error {
	if strings.TrimSpace(item.Name) == "" {
		return errText("material name is required")
	}
	return nil
}
