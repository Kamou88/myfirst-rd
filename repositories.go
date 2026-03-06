package main

import "database/sql"

type recipeRepository struct {
	db *sql.DB
}

func (r recipeRepository) List() ([]recipe, error) { return listRecipes(r.db) }
func (r recipeRepository) CreateByDeviceType(item recipe) ([]recipe, error) {
	return createRecipesByDeviceType(r.db, item)
}
func (r recipeRepository) ReplaceGroupByID(id int, item recipe) ([]recipe, bool, error) {
	return replaceRecipeGroupByID(r.db, id, item)
}
func (r recipeRepository) DeleteByID(id int) (bool, error) { return deleteRecipe(r.db, id) }
func (r recipeRepository) UpdateBooster(id int, boosterTier string) ([]recipe, bool, error) {
	return updateRecipeBoosterTier(r.db, id, boosterTier)
}

type deviceRepository struct {
	db *sql.DB
}

func (r deviceRepository) List() ([]device, error)            { return listDevices(r.db) }
func (r deviceRepository) Create(item device) (device, error) { return createDevice(r.db, item) }
func (r deviceRepository) Update(item device) (device, bool, error) {
	return updateDevice(r.db, item)
}
func (r deviceRepository) DeleteByID(id int) (bool, error) { return deleteDevice(r.db, id) }

type materialRepository struct {
	db *sql.DB
}

func (r materialRepository) List() ([]material, error) { return listMaterials(r.db) }
func (r materialRepository) Create(item material) (material, error) {
	return createMaterial(r.db, item)
}
func (r materialRepository) Update(item material) (material, bool, error) {
	return updateMaterial(r.db, item)
}
func (r materialRepository) DeleteByID(id int) (bool, error) { return deleteMaterial(r.db, id) }

type deviceTypeRepository struct {
	db *sql.DB
}

func (r deviceTypeRepository) List() ([]deviceType, error) { return listDeviceTypes(r.db) }
func (r deviceTypeRepository) Create(item deviceType) (deviceType, error) {
	return createDeviceType(r.db, item)
}
func (r deviceTypeRepository) Update(item deviceType) (deviceType, bool, error) {
	return updateDeviceType(r.db, item)
}
func (r deviceTypeRepository) DeleteByID(id int) (bool, error)  { return deleteDeviceType(r.db, id) }
func (r deviceTypeRepository) Exists(name string) (bool, error) { return deviceTypeExists(r.db, name) }
