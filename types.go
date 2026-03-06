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
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	MachineName  string           `json:"machineName"`
	DeviceModel  string           `json:"deviceModel"`
	DeviceID     int              `json:"deviceId,omitempty"`
	CycleSeconds float64          `json:"cycleSeconds"`
	PowerKW      float64          `json:"powerKW"`
	CanSpeedup   bool             `json:"canSpeedup"`
	CanBoost     bool             `json:"canBoost"`
	EffectMode   string           `json:"effectMode"`
	BoosterTier  string           `json:"boosterTier"`
	Inputs       []recipeMaterial `json:"inputs"`
	Outputs      []recipeMaterial `json:"outputs"`
}

type recipeBoosterPayload struct {
	BoosterTier string `json:"boosterTier"`
}

type device struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	DeviceType        string  `json:"deviceType"`
	EfficiencyPercent float64 `json:"efficiencyPercent"`
	PowerKW           float64 `json:"powerKW"`
}

type material struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	IsCraftable bool   `json:"isCraftable"`
	Rarity      string `json:"rarity"`
}

type deviceType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type errText string

func (e errText) Error() string {
	return string(e)
}
