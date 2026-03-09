package main

import "testing"

func TestChooseRequiredMaterialPrefersDownstreamProduct(t *testing.T) {
	recipes := []recipe{
		{
			ID:           1,
			Name:         "终产物配方",
			CycleSeconds: 60,
			PowerKW:      100,
			Outputs: []recipeMaterial{
				{Name: "目标产物", Amount: 10},
				{Name: "石墨", Amount: 3},
			},
			Inputs: []recipeMaterial{
				{Name: "原料A", Amount: 6},
			},
		},
		{
			ID:           2,
			Name:         "石墨配方",
			CycleSeconds: 60,
			PowerKW:      50,
			Outputs: []recipeMaterial{
				{Name: "石墨", Amount: 3},
			},
			Inputs: []recipeMaterial{
				{Name: "原料B", Amount: 2},
			},
		},
	}

	optionsByMaterial := buildRequirementOptionsByMaterial(recipes)
	selected, ok := chooseRequiredMaterial(
		map[string]float64{
			"石墨":   3,
			"目标产物": 10,
		},
		optionsByMaterial,
		map[string]int{},
	)
	if !ok {
		t.Fatalf("expected a selected material")
	}
	if selected.name != "目标产物" {
		t.Fatalf("expected downstream product to be selected first, got %q", selected.name)
	}
}

func TestCalculateRequirementPlanUsesByproductBeforeDedicatedRecipe(t *testing.T) {
	recipes := []recipe{
		{
			ID:                      1,
			Name:                    "终产物配方",
			DeviceModel:             "设备A",
			DeviceEfficiencyPercent: 100,
			CycleSeconds:            60,
			PowerKW:                 100,
			Outputs: []recipeMaterial{
				{Name: "目标产物", Amount: 10},
				{Name: "石墨", Amount: 3},
			},
			Inputs: []recipeMaterial{
				{Name: "原料A", Amount: 6},
			},
		},
		{
			ID:                      2,
			Name:                    "石墨配方",
			DeviceModel:             "设备B",
			DeviceEfficiencyPercent: 100,
			CycleSeconds:            60,
			PowerKW:                 50,
			Outputs: []recipeMaterial{
				{Name: "石墨", Amount: 3},
			},
			Inputs: []recipeMaterial{
				{Name: "原料B", Amount: 2},
			},
		},
	}

	result := calculateRequirementPlan(
		[]requirementTarget{
			{Name: "石墨", Amount: 3},
			{Name: "目标产物", Amount: 10},
		},
		recipes,
		"min_power",
	)

	if len(result.RecipeRows) != 1 {
		t.Fatalf("expected only one recipe row, got %d", len(result.RecipeRows))
	}
	if result.RecipeRows[0].RecipeID != 1 {
		t.Fatalf("expected final-product recipe to satisfy graphite byproduct too, got recipe %d", result.RecipeRows[0].RecipeID)
	}
}
