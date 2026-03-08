package main

import (
	"database/sql"
	"strings"
)

func createRequirementPlan(db *sql.DB, item requirementPlan) (requirementPlan, error) {
	tx, err := db.Begin()
	if err != nil {
		return requirementPlan{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.Exec(`INSERT INTO requirement_plans (name) VALUES (?)`, item.Name)
	if err != nil {
		return requirementPlan{}, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return requirementPlan{}, err
	}
	item.ID = int(id64)

	for i, target := range item.Targets {
		if _, err = tx.Exec(
			`INSERT INTO requirement_plan_targets (plan_id, name, amount, position) VALUES (?, ?, ?, ?)`,
			item.ID, target.Name, target.Amount, i,
		); err != nil {
			return requirementPlan{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		return requirementPlan{}, err
	}
	return item, nil
}

func listRequirementPlans(db *sql.DB) ([]requirementPlan, error) {
	rows, err := db.Query(`
SELECT
  p.id,
  p.name,
  t.id,
  t.name,
  t.amount
FROM requirement_plans p
LEFT JOIN requirement_plan_targets t ON t.plan_id = p.id
ORDER BY p.id ASC, t.position ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int]*requirementPlan)
	order := make([]int, 0)
	for rows.Next() {
		var (
			planID       int
			planName     string
			targetID     sql.NullInt64
			targetName   sql.NullString
			targetAmount sql.NullFloat64
		)
		if err := rows.Scan(&planID, &planName, &targetID, &targetName, &targetAmount); err != nil {
			return nil, err
		}

		plan, ok := byID[planID]
		if !ok {
			plan = &requirementPlan{
				ID:      planID,
				Name:    planName,
				Targets: make([]requirementPlanTarget, 0),
			}
			byID[planID] = plan
			order = append(order, planID)
		}

		if targetID.Valid && targetName.Valid && targetAmount.Valid {
			plan.Targets = append(plan.Targets, requirementPlanTarget{
				ID:     int(targetID.Int64),
				Name:   targetName.String,
				Amount: targetAmount.Float64,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]requirementPlan, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}

func updateRequirementPlan(db *sql.DB, item requirementPlan) (requirementPlan, bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return requirementPlan{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.Exec(`UPDATE requirement_plans SET name = ? WHERE id = ?`, item.Name, item.ID)
	if err != nil {
		return requirementPlan{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return requirementPlan{}, false, err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return requirementPlan{}, false, nil
	}

	if _, err = tx.Exec(`DELETE FROM requirement_plan_targets WHERE plan_id = ?`, item.ID); err != nil {
		return requirementPlan{}, false, err
	}
	for i, target := range item.Targets {
		if _, err = tx.Exec(
			`INSERT INTO requirement_plan_targets (plan_id, name, amount, position) VALUES (?, ?, ?, ?)`,
			item.ID, target.Name, target.Amount, i,
		); err != nil {
			return requirementPlan{}, false, err
		}
	}

	if err = tx.Commit(); err != nil {
		return requirementPlan{}, false, err
	}
	return item, true, nil
}

func deleteRequirementPlan(db *sql.DB, id int) (bool, error) {
	res, err := db.Exec(`DELETE FROM requirement_plans WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func validateRequirementPlan(item requirementPlan) error {
	if strings.TrimSpace(item.Name) == "" {
		return errText("requirement plan name is required")
	}
	if len(item.Targets) == 0 {
		return errText("at least one target is required")
	}
	for _, target := range item.Targets {
		if strings.TrimSpace(target.Name) == "" {
			return errText("target name is required")
		}
		if target.Amount <= 0 {
			return errText("target amount must be greater than 0")
		}
	}
	return nil
}
