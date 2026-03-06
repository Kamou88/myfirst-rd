package main

import (
	"database/sql"
	"strings"
)

func createDeviceType(db *sql.DB, item deviceType) (deviceType, error) {
	res, err := db.Exec(`INSERT INTO device_types (name) VALUES (?)`, item.Name)
	if err != nil {
		return deviceType{}, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return deviceType{}, err
	}
	item.ID = int(id64)
	return item, nil
}

func listDeviceTypes(db *sql.DB) ([]deviceType, error) {
	rows, err := db.Query(`
SELECT id, name
FROM device_types
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]deviceType, 0)
	for rows.Next() {
		var item deviceType
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

func updateDeviceType(db *sql.DB, item deviceType) (deviceType, bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return deviceType{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var oldName string
	if err = tx.QueryRow(`SELECT name FROM device_types WHERE id = ?`, item.ID).Scan(&oldName); err != nil {
		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			return deviceType{}, false, nil
		}
		return deviceType{}, false, err
	}

	if _, err = tx.Exec(`UPDATE device_types SET name = ? WHERE id = ?`, item.Name, item.ID); err != nil {
		return deviceType{}, false, err
	}
	if oldName != item.Name {
		if _, err = tx.Exec(`UPDATE devices SET device_type = ? WHERE device_type = ?`, item.Name, oldName); err != nil {
			return deviceType{}, false, err
		}
		if _, err = tx.Exec(`UPDATE recipes SET machine_name = ? WHERE machine_name = ?`, item.Name, oldName); err != nil {
			return deviceType{}, false, err
		}
	}

	if err = tx.Commit(); err != nil {
		return deviceType{}, false, err
	}
	return item, true, nil
}

func deleteDeviceType(db *sql.DB, id int) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var name string
	if err = tx.QueryRow(`SELECT name FROM device_types WHERE id = ?`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			return false, nil
		}
		return false, err
	}

	var deviceCount int
	if err = tx.QueryRow(`SELECT COUNT(1) FROM devices WHERE device_type = ?`, name).Scan(&deviceCount); err != nil {
		return false, err
	}
	var recipeCount int
	if err = tx.QueryRow(`SELECT COUNT(1) FROM recipes WHERE machine_name = ?`, name).Scan(&recipeCount); err != nil {
		return false, err
	}
	if deviceCount > 0 || recipeCount > 0 {
		return false, errText("device type is in use and cannot be deleted")
	}

	if _, err = tx.Exec(`DELETE FROM device_types WHERE id = ?`, id); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func deviceTypeExists(db *sql.DB, name string) (bool, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM device_types WHERE name = ?`, name).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func validateDeviceType(item deviceType) error {
	if strings.TrimSpace(item.Name) == "" {
		return errText("device type name is required")
	}
	return nil
}
