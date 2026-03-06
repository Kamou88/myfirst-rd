package main

import (
	"database/sql"
	"strings"
)

func createRecipesByDeviceType(db *sql.DB, item recipe) ([]recipe, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	devices, err := listDevicesByTypeTx(tx, item.MachineName)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, errText("no device model found for selected device type")
	}

	created := make([]recipe, 0, len(devices))
	for _, d := range devices {
		base := recipe{
			Name:         item.Name,
			MachineName:  item.MachineName,
			DeviceModel:  d.Name,
			CycleSeconds: item.CycleSeconds * 100 / d.EfficiencyPercent,
			PowerKW:      d.PowerKW,
			CanSpeedup:   item.CanSpeedup,
			CanBoost:     item.CanBoost,
			EffectMode:   "none",
			BoosterTier:  "mk3",
			Inputs:       cloneRecipeMaterials(item.Inputs),
			Outputs:      cloneRecipeMaterials(item.Outputs),
		}
		noneRecipe := applyBoosterTierToRecipe(base, "mk3")
		savedNone, insertErr := insertRecipeTx(tx, noneRecipe)
		if insertErr != nil {
			return nil, insertErr
		}
		created = append(created, savedNone)

		if base.CanSpeedup {
			speedRecipe := base
			speedRecipe.EffectMode = "speed"
			speedRecipe = applyBoosterTierToRecipe(speedRecipe, "mk3")
			saved, insertErr := insertRecipeTx(tx, speedRecipe)
			if insertErr != nil {
				return nil, insertErr
			}
			created = append(created, saved)
		}

		if base.CanBoost {
			boostRecipe := base
			boostRecipe.EffectMode = "boost"
			boostRecipe = applyBoosterTierToRecipe(boostRecipe, "mk3")
			saved, insertErr := insertRecipeTx(tx, boostRecipe)
			if insertErr != nil {
				return nil, insertErr
			}
			created = append(created, saved)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

func listDevicesByTypeTx(tx *sql.Tx, deviceType string) ([]device, error) {
	rows, err := tx.Query(`
SELECT id, name, device_type, efficiency_percent, power_kw
FROM devices
WHERE device_type = ?
ORDER BY id ASC
`, deviceType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]device, 0)
	for rows.Next() {
		var item device
		if err := rows.Scan(&item.ID, &item.Name, &item.DeviceType, &item.EfficiencyPercent, &item.PowerKW); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func replaceRecipeGroupByID(db *sql.DB, id int, item recipe) ([]recipe, bool, error) {
	var (
		oldName    string
		oldMachine string
	)
	if err := db.QueryRow(`SELECT name, machine_name FROM recipes WHERE id = ?`, id).Scan(&oldName, &oldMachine); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, false, err
	}
	if _, err := tx.Exec(`DELETE FROM recipes WHERE name = ? AND machine_name = ?`, oldName, oldMachine); err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}

	created, err := createRecipesByDeviceType(db, item)
	if err != nil {
		return nil, false, err
	}
	return created, true, nil
}

func updateRecipeBoosterTier(db *sql.DB, id int, boosterTier string) ([]recipe, bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var (
		anchorName    string
		anchorMachine string
		anchorEffect  string
	)
	if err = tx.QueryRow(`SELECT name, machine_name, effect_mode FROM recipes WHERE id = ?`, id).Scan(&anchorName, &anchorMachine, &anchorEffect); err != nil {
		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			return nil, false, nil
		}
		return nil, false, err
	}

	groupRows, err := tx.Query(`
SELECT id
FROM recipes
WHERE name = ? AND machine_name = ? AND effect_mode = ?
ORDER BY id ASC
`, anchorName, anchorMachine, anchorEffect)
	if err != nil {
		return nil, false, err
	}
	defer groupRows.Close()
	groupIDs := make([]int, 0)
	for groupRows.Next() {
		var recipeID int
		if err := groupRows.Scan(&recipeID); err != nil {
			return nil, false, err
		}
		groupIDs = append(groupIDs, recipeID)
	}
	if err := groupRows.Err(); err != nil {
		return nil, false, err
	}

	updatedItems := make([]recipe, 0, len(groupIDs))
	for _, recipeID := range groupIDs {
		updated, updateErr := updateRecipeBoosterTierTx(tx, recipeID, boosterTier)
		if updateErr != nil {
			return nil, false, updateErr
		}
		updatedItems = append(updatedItems, updated)
	}

	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return updatedItems, true, nil
}

func updateRecipeBoosterTierTx(tx *sql.Tx, id int, boosterTier string) (recipe, error) {
	var err error
	var item recipe
	var canSpeedup int
	var canBoost int
	if err = tx.QueryRow(`
SELECT id, name, machine_name, device_model, cycle_seconds, power_kw, can_speedup, can_boost, effect_mode, booster_tier
FROM recipes
WHERE id = ?
`, id).Scan(
		&item.ID,
		&item.Name,
		&item.MachineName,
		&item.DeviceModel,
		&item.CycleSeconds,
		&item.PowerKW,
		&canSpeedup,
		&canBoost,
		&item.EffectMode,
		&item.BoosterTier,
	); err != nil {
		if err == sql.ErrNoRows {
			return recipe{}, errText("recipe not found")
		}
		return recipe{}, err
	}
	item.CanSpeedup = canSpeedup != 0
	item.CanBoost = canBoost != 0

	rows, err := tx.Query(`
SELECT kind, name, amount
FROM recipe_materials
WHERE recipe_id = ?
ORDER BY kind ASC, position ASC
`, id)
	if err != nil {
		return recipe{}, err
	}
	defer rows.Close()
	item.Inputs = make([]recipeMaterial, 0)
	item.Outputs = make([]recipeMaterial, 0)
	for rows.Next() {
		var kind string
		var name string
		var amount float64
		if err := rows.Scan(&kind, &name, &amount); err != nil {
			return recipe{}, err
		}
		mat := recipeMaterial{Name: name, Amount: amount}
		if kind == "input" {
			item.Inputs = append(item.Inputs, mat)
		}
		if kind == "output" {
			item.Outputs = append(item.Outputs, mat)
		}
	}
	if err := rows.Err(); err != nil {
		return recipe{}, err
	}

	base := removeBoosterTierFromRecipe(item)
	if item.EffectMode != "none" {
		var (
			noneID          int
			noneCycle       float64
			nonePower       float64
			noneCanSpeedup  int
			noneCanBoost    int
			noneBoosterTier string
		)
		err = tx.QueryRow(`
SELECT id, cycle_seconds, power_kw, can_speedup, can_boost, booster_tier
FROM recipes
WHERE name = ? AND machine_name = ? AND device_model = ? AND effect_mode = 'none'
LIMIT 1
`, item.Name, item.MachineName, item.DeviceModel).Scan(
			&noneID,
			&noneCycle,
			&nonePower,
			&noneCanSpeedup,
			&noneCanBoost,
			&noneBoosterTier,
		)
		if err != nil && err != sql.ErrNoRows {
			return recipe{}, err
		}
		if err == nil {
			noneRows, noneErr := tx.Query(`
SELECT kind, name, amount
FROM recipe_materials
WHERE recipe_id = ?
ORDER BY kind ASC, position ASC
`, noneID)
			if noneErr != nil {
				return recipe{}, noneErr
			}
			defer noneRows.Close()

			noneInputs := make([]recipeMaterial, 0)
			noneOutputs := make([]recipeMaterial, 0)
			for noneRows.Next() {
				var kind string
				var name string
				var amount float64
				if scanErr := noneRows.Scan(&kind, &name, &amount); scanErr != nil {
					return recipe{}, scanErr
				}
				mat := recipeMaterial{Name: name, Amount: amount}
				if kind == "input" {
					noneInputs = append(noneInputs, mat)
				}
				if kind == "output" {
					noneOutputs = append(noneOutputs, mat)
				}
			}
			if noneErr = noneRows.Err(); noneErr != nil {
				return recipe{}, noneErr
			}

			base = recipe{
				ID:           item.ID,
				Name:         item.Name,
				MachineName:  item.MachineName,
				DeviceModel:  item.DeviceModel,
				CycleSeconds: noneCycle,
				PowerKW:      nonePower,
				CanSpeedup:   noneCanSpeedup != 0,
				CanBoost:     noneCanBoost != 0,
				EffectMode:   item.EffectMode,
				BoosterTier:  normalizeBoosterTier(noneBoosterTier),
				Inputs:       noneInputs,
				Outputs:      noneOutputs,
			}
		}
	}

	updated := applyBoosterTierToRecipe(base, boosterTier)
	updated.ID = id

	if _, err = tx.Exec(
		`UPDATE recipes SET cycle_seconds = ?, power_kw = ?, booster_tier = ? WHERE id = ?`,
		updated.CycleSeconds, updated.PowerKW, updated.BoosterTier, id,
	); err != nil {
		return recipe{}, err
	}
	if _, err = tx.Exec(`DELETE FROM recipe_materials WHERE recipe_id = ?`, id); err != nil {
		return recipe{}, err
	}
	for i, m := range updated.Inputs {
		if _, err = tx.Exec(
			`INSERT INTO recipe_materials (recipe_id, kind, name, amount, position) VALUES (?, 'input', ?, ?, ?)`,
			id, m.Name, m.Amount, i,
		); err != nil {
			return recipe{}, err
		}
	}
	for i, m := range updated.Outputs {
		if _, err = tx.Exec(
			`INSERT INTO recipe_materials (recipe_id, kind, name, amount, position) VALUES (?, 'output', ?, ?, ?)`,
			id, m.Name, m.Amount, i,
		); err != nil {
			return recipe{}, err
		}
	}
	return updated, nil
}

func insertRecipeTx(tx *sql.Tx, item recipe) (recipe, error) {
	res, err := tx.Exec(
		`INSERT INTO recipes (name, machine_name, device_model, cycle_seconds, power_kw, can_speedup, can_boost, effect_mode, booster_tier) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Name, item.MachineName, item.DeviceModel, item.CycleSeconds, item.PowerKW, boolToInt(item.CanSpeedup), boolToInt(item.CanBoost), item.EffectMode, normalizeBoosterTier(item.BoosterTier),
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
	return item, nil
}

func listRecipes(db *sql.DB) ([]recipe, error) {
	rows, err := db.Query(`
SELECT
  r.id,
  r.name,
  r.machine_name,
  r.device_model,
  r.cycle_seconds,
  r.power_kw,
  r.can_speedup,
  r.can_boost,
  r.effect_mode,
  r.booster_tier,
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
			deviceModel  string
			cycleSeconds float64
			powerKW      float64
			canSpeedup   int
			canBoost     int
			effectMode   string
			boosterTier  string
			kind         sql.NullString
			materialName sql.NullString
			amount       sql.NullFloat64
		)
		if err := rows.Scan(&id, &name, &machineName, &deviceModel, &cycleSeconds, &powerKW, &canSpeedup, &canBoost, &effectMode, &boosterTier, &kind, &materialName, &amount); err != nil {
			return nil, err
		}

		item, ok := byID[id]
		if !ok {
			item = &recipe{
				ID:           id,
				Name:         name,
				MachineName:  machineName,
				DeviceModel:  deviceModel,
				CycleSeconds: cycleSeconds,
				PowerKW:      powerKW,
				CanSpeedup:   canSpeedup != 0,
				CanBoost:     canBoost != 0,
				EffectMode:   effectMode,
				BoosterTier:  normalizeBoosterTier(boosterTier),
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

func updateRecipe(db *sql.DB, item recipe) (recipe, bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return recipe{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if strings.TrimSpace(item.EffectMode) == "" {
		if err := tx.QueryRow(`SELECT effect_mode FROM recipes WHERE id = ?`, item.ID).Scan(&item.EffectMode); err != nil {
			if err == sql.ErrNoRows {
				_ = tx.Rollback()
				return recipe{}, false, nil
			}
			return recipe{}, false, err
		}
	}
	if strings.TrimSpace(item.BoosterTier) == "" {
		if err := tx.QueryRow(`SELECT booster_tier FROM recipes WHERE id = ?`, item.ID).Scan(&item.BoosterTier); err != nil {
			if err == sql.ErrNoRows {
				_ = tx.Rollback()
				return recipe{}, false, nil
			}
			return recipe{}, false, err
		}
	}

	res, err := tx.Exec(
		`UPDATE recipes SET name = ?, machine_name = ?, cycle_seconds = ?, power_kw = ?, can_speedup = ?, can_boost = ?, effect_mode = ?, booster_tier = ? WHERE id = ?`,
		item.Name, item.MachineName, item.CycleSeconds, item.PowerKW, boolToInt(item.CanSpeedup), boolToInt(item.CanBoost), item.EffectMode, normalizeBoosterTier(item.BoosterTier), item.ID,
	)
	if err != nil {
		return recipe{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return recipe{}, false, err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return recipe{}, false, nil
	}

	if _, err = tx.Exec(`DELETE FROM recipe_materials WHERE recipe_id = ?`, item.ID); err != nil {
		return recipe{}, false, err
	}
	for i, m := range item.Inputs {
		_, err = tx.Exec(
			`INSERT INTO recipe_materials (recipe_id, kind, name, amount, position) VALUES (?, 'input', ?, ?, ?)`,
			item.ID, m.Name, m.Amount, i,
		)
		if err != nil {
			return recipe{}, false, err
		}
	}
	for i, m := range item.Outputs {
		_, err = tx.Exec(
			`INSERT INTO recipe_materials (recipe_id, kind, name, amount, position) VALUES (?, 'output', ?, ?, ?)`,
			item.ID, m.Name, m.Amount, i,
		)
		if err != nil {
			return recipe{}, false, err
		}
	}

	if err = tx.Commit(); err != nil {
		return recipe{}, false, err
	}
	return item, true, nil
}

func deleteRecipe(db *sql.DB, id int) (bool, error) {
	res, err := db.Exec(`DELETE FROM recipes WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func normalizeBoosterTier(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "mk1":
		return "mk1"
	case "mk2":
		return "mk2"
	default:
		return "mk3"
	}
}

func validateBoosterTier(tier string) error {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "mk1", "mk2", "mk3":
		return nil
	default:
		return errText("boosterTier must be one of mk1, mk2, mk3")
	}
}

func boosterMultipliers(tier string, effectMode string) (float64, float64, float64) {
	mode := strings.ToLower(strings.TrimSpace(effectMode))
	switch normalizeBoosterTier(tier) {
	case "mk1":
		switch mode {
		case "speed":
			return 0.75, 1, 1.3
		case "boost":
			return 1, 1.125, 1.3
		default:
			return 1, 1, 1
		}
	case "mk2":
		switch mode {
		case "speed":
			return 2.0 / 3.0, 1, 1.7
		case "boost":
			return 1, 1.2, 1.7
		default:
			return 1, 1, 1
		}
	default:
		switch mode {
		case "speed":
			return 0.5, 1, 2.5
		case "boost":
			return 1, 1.25, 2.5
		default:
			return 1, 1, 1
		}
	}
}

func applyBoosterTierToRecipe(item recipe, tier string) recipe {
	result := item
	result.BoosterTier = normalizeBoosterTier(tier)
	cycleMultiplier, outputMultiplier, powerMultiplier := boosterMultipliers(result.BoosterTier, result.EffectMode)
	result.CycleSeconds = result.CycleSeconds * cycleMultiplier
	result.PowerKW = result.PowerKW * powerMultiplier
	result.Outputs = cloneRecipeMaterials(result.Outputs)
	for i := range result.Outputs {
		result.Outputs[i].Amount = result.Outputs[i].Amount * outputMultiplier
	}
	return result
}

func removeBoosterTierFromRecipe(item recipe) recipe {
	result := item
	cycleMultiplier, outputMultiplier, powerMultiplier := boosterMultipliers(result.BoosterTier, result.EffectMode)
	if cycleMultiplier == 0 || powerMultiplier == 0 || outputMultiplier == 0 {
		return result
	}
	result.CycleSeconds = result.CycleSeconds / cycleMultiplier
	result.PowerKW = result.PowerKW / powerMultiplier
	result.Outputs = cloneRecipeMaterials(result.Outputs)
	for i := range result.Outputs {
		result.Outputs[i].Amount = result.Outputs[i].Amount / outputMultiplier
	}
	return result
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
	if strings.TrimSpace(item.BoosterTier) != "" {
		if err := validateBoosterTier(item.BoosterTier); err != nil {
			return err
		}
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

func cloneRecipeMaterials(items []recipeMaterial) []recipeMaterial {
	out := make([]recipeMaterial, len(items))
	copy(out, items)
	return out
}
