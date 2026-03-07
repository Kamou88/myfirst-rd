package main

import (
	"database/sql"
	"strings"
)

func createMaterial(db *sql.DB, item material) (material, error) {
	res, err := db.Exec(
		`INSERT INTO materials (name, is_craftable, is_raw, rarity) VALUES (?, ?, ?, ?)`,
		item.Name,
		boolToInt(item.IsCraftable),
		boolToInt(item.IsRaw),
		normalizeMaterialRarity(item.Rarity),
	)
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
SELECT id, name, is_craftable, is_raw, rarity
FROM materials
ORDER BY
  CASE rarity
    WHEN '一般' THEN 1
    WHEN '普通' THEN 2
    WHEN '稀有' THEN 3
    WHEN '史诗' THEN 4
    WHEN '传说' THEN 5
    ELSE 99
  END ASC,
  LENGTH(name) ASC,
  name COLLATE NOCASE ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]material, 0)
	for rows.Next() {
		var item material
		var isCraftable int
		var isRaw int
		if err := rows.Scan(&item.ID, &item.Name, &isCraftable, &isRaw, &item.Rarity); err != nil {
			return nil, err
		}
		item.IsCraftable = isCraftable != 0
		item.IsRaw = isRaw != 0
		item.Rarity = normalizeMaterialRarity(item.Rarity)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func updateMaterial(db *sql.DB, item material) (material, bool, error) {
	res, err := db.Exec(
		`UPDATE materials SET name = ?, is_craftable = ?, is_raw = ?, rarity = ? WHERE id = ?`,
		item.Name, boolToInt(item.IsCraftable), boolToInt(item.IsRaw), normalizeMaterialRarity(item.Rarity), item.ID,
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

func syncMaterialRawFromRecipes(db *sql.DB) error {
	return syncMaterialRawByRecipeInputs(db)
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
	if err := validateMaterialRarity(item.Rarity); err != nil {
		return err
	}
	return nil
}

func normalizeMaterialRarity(rarity string) string {
	switch strings.TrimSpace(rarity) {
	case "一般", "普通", "稀有", "史诗", "传说":
		return strings.TrimSpace(rarity)
	default:
		return "一般"
	}
}

func validateMaterialRarity(rarity string) error {
	switch strings.TrimSpace(rarity) {
	case "", "一般", "普通", "稀有", "史诗", "传说":
		return nil
	default:
		return errText("rarity must be one of 一般, 普通, 稀有, 史诗, 传说")
	}
}
