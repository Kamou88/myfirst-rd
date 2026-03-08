package main

import (
	"database/sql"
	"strings"
)

func createProductionLine(db *sql.DB, item productionLine) (productionLine, error) {
	tx, err := db.Begin()
	if err != nil {
		return productionLine{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.Exec(`INSERT INTO production_lines (name) VALUES (?)`, item.Name)
	if err != nil {
		return productionLine{}, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return productionLine{}, err
	}
	item.ID = int(id64)

	if err = ensureResearchedRecipesTx(tx, item.Items); err != nil {
		return productionLine{}, err
	}

	for i, lineItem := range item.Items {
		if _, err = tx.Exec(
			`INSERT INTO production_line_items (line_id, recipe_id, machine_count, position) VALUES (?, ?, ?, ?)`,
			item.ID, lineItem.RecipeID, lineItem.MachineCount, i,
		); err != nil {
			return productionLine{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		return productionLine{}, err
	}
	return item, nil
}

func listProductionLines(db *sql.DB) ([]productionLine, error) {
	rows, err := db.Query(`
SELECT
  l.id,
  l.name,
  i.id,
  i.recipe_id,
  i.machine_count
FROM production_lines l
LEFT JOIN production_line_items i ON i.line_id = l.id
ORDER BY l.id ASC, i.position ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int]*productionLine)
	order := make([]int, 0)
	for rows.Next() {
		var (
			lineID       int
			lineName     string
			itemID       sql.NullInt64
			recipeID     sql.NullInt64
			machineCount sql.NullInt64
		)
		if err := rows.Scan(&lineID, &lineName, &itemID, &recipeID, &machineCount); err != nil {
			return nil, err
		}

		line, ok := byID[lineID]
		if !ok {
			line = &productionLine{
				ID:    lineID,
				Name:  lineName,
				Items: make([]productionLineItem, 0),
			}
			byID[lineID] = line
			order = append(order, lineID)
		}

		if itemID.Valid && recipeID.Valid && machineCount.Valid {
			line.Items = append(line.Items, productionLineItem{
				ID:           int(itemID.Int64),
				RecipeID:     int(recipeID.Int64),
				MachineCount: int(machineCount.Int64),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]productionLine, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}

func updateProductionLine(db *sql.DB, item productionLine) (productionLine, bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return productionLine{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.Exec(`UPDATE production_lines SET name = ? WHERE id = ?`, item.Name, item.ID)
	if err != nil {
		return productionLine{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return productionLine{}, false, err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return productionLine{}, false, nil
	}

	if _, err = tx.Exec(`DELETE FROM production_line_items WHERE line_id = ?`, item.ID); err != nil {
		return productionLine{}, false, err
	}

	if err = ensureResearchedRecipesTx(tx, item.Items); err != nil {
		return productionLine{}, false, err
	}

	for i, lineItem := range item.Items {
		if _, err = tx.Exec(
			`INSERT INTO production_line_items (line_id, recipe_id, machine_count, position) VALUES (?, ?, ?, ?)`,
			item.ID, lineItem.RecipeID, lineItem.MachineCount, i,
		); err != nil {
			return productionLine{}, false, err
		}
	}

	if err = tx.Commit(); err != nil {
		return productionLine{}, false, err
	}
	return item, true, nil
}

func deleteProductionLine(db *sql.DB, id int) (bool, error) {
	res, err := db.Exec(`DELETE FROM production_lines WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func validateProductionLine(item productionLine) error {
	if strings.TrimSpace(item.Name) == "" {
		return errText("production line name is required")
	}
	if len(item.Items) == 0 {
		return errText("at least one production line item is required")
	}
	for _, lineItem := range item.Items {
		if lineItem.RecipeID <= 0 {
			return errText("recipeId must be greater than 0")
		}
		if lineItem.MachineCount <= 0 {
			return errText("machineCount must be greater than 0")
		}
	}
	return nil
}

func ensureResearchedRecipesTx(tx *sql.Tx, items []productionLineItem) error {
	checked := make(map[int]struct{})
	for _, lineItem := range items {
		if _, ok := checked[lineItem.RecipeID]; ok {
			continue
		}
		checked[lineItem.RecipeID] = struct{}{}

		var (
			isResearched   int
			deviceUnlocked int
		)
		if err := tx.QueryRow(`
SELECT r.is_researched, COALESCE(d.is_unlocked, 0)
FROM recipes r
LEFT JOIN devices d ON d.id = r.device_id
WHERE r.id = ?
`, lineItem.RecipeID).Scan(&isResearched, &deviceUnlocked); err != nil {
			if err == sql.ErrNoRows {
				return errText("recipe not found")
			}
			return err
		}
		if isResearched == 0 {
			return errText("only researched recipes can be selected")
		}
		if deviceUnlocked == 0 {
			return errText("only recipes with unlocked devices can be selected")
		}
	}
	return nil
}
