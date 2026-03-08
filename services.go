package main

import "strings"

type recipeService struct {
	repo recipeRepository
}

func (s recipeService) List() ([]recipe, error) { return s.repo.List() }

func (s recipeService) Create(payload recipe) ([]recipe, error) {
	if err := validateRecipe(payload); err != nil {
		return nil, err
	}
	return s.repo.CreateByDeviceType(recipe{
		Name:         strings.TrimSpace(payload.Name),
		MachineName:  strings.TrimSpace(payload.MachineName),
		CycleSeconds: payload.CycleSeconds,
		PowerKW:      payload.PowerKW,
		CanSpeedup:   payload.CanSpeedup,
		CanBoost:     payload.CanBoost,
		IsResearched: payload.IsResearched,
		EffectMode:   "",
		Inputs:       sanitizeMaterials(payload.Inputs),
		Outputs:      sanitizeMaterials(payload.Outputs),
	})
}

func (s recipeService) ReplaceGroup(id int, payload recipe) ([]recipe, bool, error) {
	payload.ID = id
	if err := validateRecipe(payload); err != nil {
		return nil, false, err
	}
	return s.repo.ReplaceGroupByID(id, recipe{
		ID:           id,
		Name:         strings.TrimSpace(payload.Name),
		MachineName:  strings.TrimSpace(payload.MachineName),
		CycleSeconds: payload.CycleSeconds,
		PowerKW:      payload.PowerKW,
		CanSpeedup:   payload.CanSpeedup,
		CanBoost:     payload.CanBoost,
		IsResearched: payload.IsResearched,
		EffectMode:   "",
		Inputs:       sanitizeMaterials(payload.Inputs),
		Outputs:      sanitizeMaterials(payload.Outputs),
	})
}

func (s recipeService) Delete(id int) (bool, error) { return s.repo.DeleteByID(id) }

func (s recipeService) UpdateBooster(id int, boosterTier string) ([]recipe, bool, error) {
	if err := validateBoosterTier(boosterTier); err != nil {
		return nil, false, err
	}
	return s.repo.UpdateBooster(id, boosterTier)
}

func (s recipeService) UpdateResearch(id int, isResearched bool) (bool, error) {
	return s.repo.UpdateResearch(id, isResearched)
}

func (s recipeService) CalculateRequirements(payload requirementCalculatePayload) (requirementCalculateResponse, error) {
	allRecipes, err := s.repo.List()
	if err != nil {
		return requirementCalculateResponse{}, err
	}

	eligibleRecipes := make([]recipe, 0, len(allRecipes))
	for _, item := range allRecipes {
		if item.IsResearched && item.DeviceUnlocked {
			eligibleRecipes = append(eligibleRecipes, item)
		}
	}
	if len(eligibleRecipes) == 0 {
		return requirementCalculateResponse{}, errText("当前没有“已研究且设备已解锁”的配方，无法参与需求计算")
	}

	return requirementCalculateResponse{
		MinPower: calculateRequirementPlan(payload.Targets, eligibleRecipes, "min_power"),
		MinRaw:   calculateRequirementPlan(payload.Targets, eligibleRecipes, "min_raw"),
	}, nil
}

type deviceService struct {
	repo     deviceRepository
	typeRepo deviceTypeRepository
}

func (s deviceService) List() ([]device, error) { return s.repo.List() }

func (s deviceService) Create(payload device) (device, error) {
	if err := validateDevice(payload); err != nil {
		return device{}, err
	}
	exists, err := s.typeRepo.Exists(strings.TrimSpace(payload.DeviceType))
	if err != nil {
		return device{}, err
	}
	if !exists {
		return device{}, errText("deviceType does not exist, please create it first")
	}
	return s.repo.Create(device{
		Name:              strings.TrimSpace(payload.Name),
		DeviceType:        strings.TrimSpace(payload.DeviceType),
		EfficiencyPercent: payload.EfficiencyPercent,
		PowerKW:           payload.PowerKW,
		IsUnlocked:        payload.IsUnlocked,
	})
}

func (s deviceService) Update(id int, payload device) (device, bool, error) {
	payload.ID = id
	if err := validateDevice(payload); err != nil {
		return device{}, false, err
	}
	exists, err := s.typeRepo.Exists(strings.TrimSpace(payload.DeviceType))
	if err != nil {
		return device{}, false, err
	}
	if !exists {
		return device{}, false, errText("deviceType does not exist, please create it first")
	}
	return s.repo.Update(payload)
}

func (s deviceService) Delete(id int) (bool, error) { return s.repo.DeleteByID(id) }
func (s deviceService) UpdateUnlock(id int, isUnlocked bool) (bool, error) {
	return s.repo.UpdateUnlock(id, isUnlocked)
}

type materialService struct {
	repo materialRepository
}

func (s materialService) List() ([]material, error) { return s.repo.List() }
func (s materialService) Create(payload material) (material, error) {
	if err := validateMaterial(payload); err != nil {
		return material{}, err
	}
	return s.repo.Create(material{
		Name:        strings.TrimSpace(payload.Name),
		IsCraftable: payload.IsCraftable,
		IsRaw:       payload.IsRaw,
		Rarity:      normalizeMaterialRarity(payload.Rarity),
	})
}
func (s materialService) Update(id int, payload material) (material, bool, error) {
	payload.ID = id
	if err := validateMaterial(payload); err != nil {
		return material{}, false, err
	}
	payload.Rarity = normalizeMaterialRarity(payload.Rarity)
	return s.repo.Update(payload)
}
func (s materialService) Delete(id int) (bool, error) { return s.repo.DeleteByID(id) }
func (s materialService) SyncRawByRecipeInputs() ([]material, error) {
	if err := s.repo.SyncRawByRecipeInputs(); err != nil {
		return nil, err
	}
	return s.repo.List()
}

type deviceTypeService struct {
	repo deviceTypeRepository
}

func (s deviceTypeService) List() ([]deviceType, error) { return s.repo.List() }
func (s deviceTypeService) Create(payload deviceType) (deviceType, error) {
	if err := validateDeviceType(payload); err != nil {
		return deviceType{}, err
	}
	return s.repo.Create(deviceType{Name: strings.TrimSpace(payload.Name)})
}
func (s deviceTypeService) Update(id int, payload deviceType) (deviceType, bool, error) {
	payload.ID = id
	if err := validateDeviceType(payload); err != nil {
		return deviceType{}, false, err
	}
	return s.repo.Update(payload)
}
func (s deviceTypeService) Delete(id int) (bool, error) { return s.repo.DeleteByID(id) }

type productionLineService struct {
	repo productionLineRepository
}

func (s productionLineService) List() ([]productionLine, error) { return s.repo.List() }
func (s productionLineService) Create(payload productionLine) (productionLine, error) {
	if err := validateProductionLine(payload); err != nil {
		return productionLine{}, err
	}
	items := make([]productionLineItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, productionLineItem{
			RecipeID:     item.RecipeID,
			MachineCount: item.MachineCount,
		})
	}
	return s.repo.Create(productionLine{
		Name:  strings.TrimSpace(payload.Name),
		Items: items,
	})
}
func (s productionLineService) Update(id int, payload productionLine) (productionLine, bool, error) {
	payload.ID = id
	if err := validateProductionLine(payload); err != nil {
		return productionLine{}, false, err
	}
	items := make([]productionLineItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, productionLineItem{
			RecipeID:     item.RecipeID,
			MachineCount: item.MachineCount,
		})
	}
	return s.repo.Update(productionLine{
		ID:    id,
		Name:  strings.TrimSpace(payload.Name),
		Items: items,
	})
}
func (s productionLineService) Delete(id int) (bool, error) { return s.repo.DeleteByID(id) }

type requirementPlanService struct {
	repo requirementPlanRepository
}

func (s requirementPlanService) List() ([]requirementPlan, error) { return s.repo.List() }
func (s requirementPlanService) Create(payload requirementPlan) (requirementPlan, error) {
	if err := validateRequirementPlan(payload); err != nil {
		return requirementPlan{}, err
	}
	targets := make([]requirementPlanTarget, 0, len(payload.Targets))
	for _, target := range payload.Targets {
		targets = append(targets, requirementPlanTarget{
			Name:   strings.TrimSpace(target.Name),
			Amount: target.Amount,
		})
	}
	return s.repo.Create(requirementPlan{
		Name:    strings.TrimSpace(payload.Name),
		Targets: targets,
	})
}
func (s requirementPlanService) Update(id int, payload requirementPlan) (requirementPlan, bool, error) {
	payload.ID = id
	if err := validateRequirementPlan(payload); err != nil {
		return requirementPlan{}, false, err
	}
	targets := make([]requirementPlanTarget, 0, len(payload.Targets))
	for _, target := range payload.Targets {
		targets = append(targets, requirementPlanTarget{
			Name:   strings.TrimSpace(target.Name),
			Amount: target.Amount,
		})
	}
	return s.repo.Update(requirementPlan{
		ID:      id,
		Name:    strings.TrimSpace(payload.Name),
		Targets: targets,
	})
}
func (s requirementPlanService) Delete(id int) (bool, error) { return s.repo.DeleteByID(id) }

type appServices struct {
	recipes          recipeService
	devices          deviceService
	materials        materialService
	deviceTypes      deviceTypeService
	linePlans        productionLineService
	requirementPlans requirementPlanService
}
