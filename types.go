package main

type healthResponse struct {
	Message string `json:"message"`
}

type recipeMaterial struct {
	MaterialID int     `json:"materialId,omitempty"`
	Name       string  `json:"name"`
	Amount     float64 `json:"amount"`
}

type recipe struct {
	ID                      int              `json:"id"`
	Name                    string           `json:"name"`
	MachineName             string           `json:"machineName"`
	DeviceModel             string           `json:"deviceModel"`
	DeviceID                int              `json:"deviceId,omitempty"`
	DeviceUnlocked          bool             `json:"deviceUnlocked"`
	DeviceEfficiencyPercent float64          `json:"deviceEfficiencyPercent"`
	CycleSeconds            float64          `json:"cycleSeconds"`
	PowerKW                 float64          `json:"powerKW"`
	CanSpeedup              bool             `json:"canSpeedup"`
	CanBoost                bool             `json:"canBoost"`
	IsResearched            bool             `json:"isResearched"`
	EffectMode              string           `json:"effectMode"`
	BoosterTier             string           `json:"boosterTier"`
	Inputs                  []recipeMaterial `json:"inputs"`
	Outputs                 []recipeMaterial `json:"outputs"`
}

type recipeBoosterPayload struct {
	BoosterTier string `json:"boosterTier"`
}

type recipeResearchPayload struct {
	IsResearched bool `json:"isResearched"`
}

type requirementTarget struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type requirementCalculatePayload struct {
	Targets []requirementTarget `json:"targets"`
}

type requirementMaterialAmount struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type requirementRecipeRow struct {
	RecipeID     int     `json:"recipeID"`
	RecipeName   string  `json:"recipeName"`
	DeviceModel  string  `json:"deviceModel"`
	EffectMode   string  `json:"effectMode"`
	MachineCount int     `json:"machineCount"`
	PowerKW      float64 `json:"powerKW"`
}

type requirementPlanResult struct {
	RecipeRows           []requirementRecipeRow      `json:"recipeRows"`
	ExternalInputs       []requirementMaterialAmount `json:"externalInputs"`
	UnresolvedCraftables []requirementMaterialAmount `json:"unresolvedCraftables"`
	ActualOutputs        []requirementMaterialAmount `json:"actualOutputs"`
	ActualInputs         []requirementMaterialAmount `json:"actualInputs"`
	TotalPowerKW         float64                     `json:"totalPowerKW"`
	TotalExternalInputs  float64                     `json:"totalExternalInputs"`
	Warnings             []string                    `json:"warnings"`
}

type requirementCalculateResponse struct {
	MinPower requirementPlanResult `json:"minPower"`
	MinRaw   requirementPlanResult `json:"minRaw"`
}

type device struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	DeviceType        string  `json:"deviceType"`
	EfficiencyPercent float64 `json:"efficiencyPercent"`
	PowerKW           float64 `json:"powerKW"`
	IsUnlocked        bool    `json:"isUnlocked"`
}

type deviceUnlockPayload struct {
	IsUnlocked bool `json:"isUnlocked"`
}

type material struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	IsCraftable bool   `json:"isCraftable"`
	IsRaw       bool   `json:"isRaw"`
	Rarity      string `json:"rarity"`
}

type deviceType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type productionLineItem struct {
	ID           int `json:"id"`
	RecipeID     int `json:"recipeId"`
	MachineCount int `json:"machineCount"`
}

type productionLine struct {
	ID    int                  `json:"id"`
	Name  string               `json:"name"`
	Items []productionLineItem `json:"items"`
}

type errText string

func (e errText) Error() string {
	return string(e)
}
