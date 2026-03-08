package main

import (
	"database/sql"
	"strings"
)

func createDevice(db *sql.DB, item device) (device, error) {
	res, err := db.Exec(
		`INSERT INTO devices (name, device_type, efficiency_percent, power_kw, is_unlocked) VALUES (?, ?, ?, ?, ?)`,
		item.Name, item.DeviceType, item.EfficiencyPercent, item.PowerKW, boolToInt(item.IsUnlocked),
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
SELECT id, name, device_type, efficiency_percent, power_kw, is_unlocked
FROM devices
ORDER BY device_type ASC, efficiency_percent ASC, id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]device, 0)
	for rows.Next() {
		var item device
		var isUnlocked int
		if err := rows.Scan(&item.ID, &item.Name, &item.DeviceType, &item.EfficiencyPercent, &item.PowerKW, &isUnlocked); err != nil {
			return nil, err
		}
		item.IsUnlocked = isUnlocked != 0
		if strings.TrimSpace(item.DeviceType) == "" {
			item.DeviceType = "未分类"
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func updateDevice(db *sql.DB, item device) (device, bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return device{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var oldItem device
	var oldUnlocked int
	if err = tx.QueryRow(
		`SELECT id, name, device_type, efficiency_percent, power_kw, is_unlocked FROM devices WHERE id = ?`,
		item.ID,
	).Scan(&oldItem.ID, &oldItem.Name, &oldItem.DeviceType, &oldItem.EfficiencyPercent, &oldItem.PowerKW, &oldUnlocked); err != nil {
		if err == sql.ErrNoRows {
			return device{}, false, nil
		}
		return device{}, false, err
	}
	oldItem.IsUnlocked = oldUnlocked != 0

	res, err := tx.Exec(
		`UPDATE devices SET name = ?, device_type = ?, efficiency_percent = ?, power_kw = ?, is_unlocked = ? WHERE id = ?`,
		item.Name, item.DeviceType, item.EfficiencyPercent, item.PowerKW, boolToInt(item.IsUnlocked), item.ID,
	)
	if err != nil {
		return device{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return device{}, false, err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return device{}, false, nil
	}

	if err = syncRecipesForDeviceTx(tx, oldItem, item); err != nil {
		return device{}, false, err
	}

	if err = tx.Commit(); err != nil {
		return device{}, false, err
	}

	return item, affected > 0, nil
}

func updateDeviceUnlockStatus(db *sql.DB, id int, isUnlocked bool) (bool, error) {
	res, err := db.Exec(`UPDATE devices SET is_unlocked = ? WHERE id = ?`, boolToInt(isUnlocked), id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
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

func validateDevice(item device) error {
	if strings.TrimSpace(item.Name) == "" {
		return errText("device model name is required")
	}
	if strings.TrimSpace(item.DeviceType) == "" {
		return errText("deviceType is required")
	}
	if item.EfficiencyPercent <= 0 {
		return errText("efficiencyPercent must be greater than 0")
	}
	if item.PowerKW < 0 {
		return errText("powerKW cannot be negative")
	}
	return nil
}
