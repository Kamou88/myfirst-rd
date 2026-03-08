package main

import (
	"sort"
	"strings"
)

const (
	requirementEPS      = 1e-6
	requirementMaxSteps = 3000
)

type requirementOption struct {
	recipe         recipe
	outRate        float64
	totalInputRate float64
	powerRate      float64
	powerPerOut    float64
	inputPerOut    float64
}

type machineAllocation struct {
	option requirementOption
	count  int
}

func perMinute(amount float64, cycleSeconds float64) float64 {
	if cycleSeconds <= 0 {
		return 0
	}
	return amount * 60 / cycleSeconds
}

func buildRequirementOptionsByMaterial(recipes []recipe) map[string][]requirementOption {
	optionsByMaterial := make(map[string][]requirementOption)
	for _, item := range recipes {
		for _, output := range item.Outputs {
			material := strings.TrimSpace(output.Name)
			outRate := perMinute(output.Amount, item.CycleSeconds)
			if material == "" || outRate <= requirementEPS {
				continue
			}

			totalInputRate := 0.0
			for _, input := range item.Inputs {
				totalInputRate += perMinute(input.Amount, item.CycleSeconds)
			}
			powerRate := item.PowerKW
			option := requirementOption{
				recipe:         item,
				outRate:        outRate,
				totalInputRate: totalInputRate,
				powerRate:      powerRate,
				powerPerOut:    powerRate / outRate,
				inputPerOut:    totalInputRate / outRate,
			}
			optionsByMaterial[material] = append(optionsByMaterial[material], option)
		}
	}
	return optionsByMaterial
}

func chooseRequirementOption(options []requirementOption, strategy string) (requirementOption, bool) {
	if len(options) == 0 {
		return requirementOption{}, false
	}
	sorted := make([]requirementOption, len(options))
	copy(sorted, options)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		// Prefer higher-efficiency devices before applying strategy-specific costs.
		if a.recipe.DeviceEfficiencyPercent != b.recipe.DeviceEfficiencyPercent {
			return a.recipe.DeviceEfficiencyPercent > b.recipe.DeviceEfficiencyPercent
		}
		if strategy == "min_power" {
			if a.powerPerOut != b.powerPerOut {
				return a.powerPerOut < b.powerPerOut
			}
			if a.inputPerOut != b.inputPerOut {
				return a.inputPerOut < b.inputPerOut
			}
			return a.outRate > b.outRate
		}
		if a.inputPerOut != b.inputPerOut {
			return a.inputPerOut < b.inputPerOut
		}
		if a.powerPerOut != b.powerPerOut {
			return a.powerPerOut < b.powerPerOut
		}
		return a.outRate > b.outRate
	})
	return sorted[0], true
}

func ceilMachineCount(need float64, outRate float64) int {
	if outRate <= requirementEPS {
		return 0
	}
	count := int(need / outRate)
	if float64(count)*outRate < need-requirementEPS {
		count++
	}
	if count < 1 {
		count = 1
	}
	return count
}

func allocationProduced(allocations []machineAllocation) float64 {
	total := 0.0
	for _, item := range allocations {
		total += float64(item.count) * item.option.outRate
	}
	return total
}

func allocationCost(allocations []machineAllocation, strategy string) float64 {
	total := 0.0
	for _, item := range allocations {
		if strategy == "min_power" {
			total += float64(item.count) * item.option.powerRate
		} else {
			total += float64(item.count) * item.option.totalInputRate
		}
	}
	return total
}

func buildIntegerAllocations(
	need float64,
	picked requirementOption,
	allOptions []requirementOption,
	strategy string,
) []machineAllocation {
	baseCount := ceilMachineCount(need, picked.outRate)
	best := []machineAllocation{{option: picked, count: baseCount}}
	bestOverflow := allocationProduced(best) - need
	bestCost := allocationCost(best, strategy)

	for _, low := range allOptions {
		if low.recipe.ID == picked.recipe.ID {
			continue
		}
		// Replace with lower-efficiency device recipes to reduce overflow.
		if low.recipe.DeviceEfficiencyPercent >= picked.recipe.DeviceEfficiencyPercent {
			continue
		}
		if low.outRate <= requirementEPS {
			continue
		}

		for replaced := 1; replaced < baseCount; replaced++ {
			highCount := baseCount - replaced
			remainingNeed := need - float64(highCount)*picked.outRate
			if remainingNeed <= requirementEPS {
				continue
			}
			lowCount := ceilMachineCount(remainingNeed, low.outRate)
			if lowCount <= 0 {
				continue
			}

			trial := make([]machineAllocation, 0, 2)
			if highCount > 0 {
				trial = append(trial, machineAllocation{option: picked, count: highCount})
			}
			trial = append(trial, machineAllocation{option: low, count: lowCount})

			overflow := allocationProduced(trial) - need
			cost := allocationCost(trial, strategy)
			if overflow < bestOverflow-requirementEPS ||
				(abs(overflow-bestOverflow) <= requirementEPS && cost < bestCost-requirementEPS) {
				best = trial
				bestOverflow = overflow
				bestCost = cost
			}
		}
	}
	return best
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func calculateRequirementPlan(targets []requirementTarget, recipes []recipe, strategy string) requirementPlanResult {
	requirement := make(map[string]float64)
	for _, target := range targets {
		name := strings.TrimSpace(target.Name)
		if name == "" || target.Amount <= requirementEPS {
			continue
		}
		requirement[name] += target.Amount
	}

	optionsByMaterial := buildRequirementOptionsByMaterial(recipes)
	machineByRecipeID := make(map[int]int)
	recipeByID := make(map[int]recipe, len(recipes))
	for _, item := range recipes {
		recipeByID[item.ID] = item
	}
	warnings := make([]string, 0)

	steps := 0
	for steps < requirementMaxSteps {
		steps++
		selectedMaterial := ""
		selectedNeed := 0.0
		for name, value := range requirement {
			if value > requirementEPS {
				if _, ok := optionsByMaterial[name]; ok {
					selectedMaterial = name
					selectedNeed = value
					break
				}
			}
		}
		if selectedMaterial == "" {
			break
		}

		picked, ok := chooseRequirementOption(optionsByMaterial[selectedMaterial], strategy)
		if !ok || picked.outRate <= requirementEPS {
			break
		}
		allocations := buildIntegerAllocations(
			selectedNeed,
			picked,
			optionsByMaterial[selectedMaterial],
			strategy,
		)
		for _, alloc := range allocations {
			if alloc.count <= 0 {
				continue
			}
			machineByRecipeID[alloc.option.recipe.ID] += alloc.count
			for _, output := range alloc.option.recipe.Outputs {
				outputRate := perMinute(output.Amount, alloc.option.recipe.CycleSeconds)
				requirement[output.Name] = requirement[output.Name] - outputRate*float64(alloc.count)
			}
			for _, input := range alloc.option.recipe.Inputs {
				inputRate := perMinute(input.Amount, alloc.option.recipe.CycleSeconds)
				requirement[input.Name] = requirement[input.Name] + inputRate*float64(alloc.count)
			}
		}
	}

	if steps >= requirementMaxSteps {
		warnings = append(warnings, "计算步数达到上限，可能存在循环依赖，请检查配方关系。")
	}

	recipeRows := make([]requirementRecipeRow, 0, len(machineByRecipeID))
	totalPowerKW := 0.0
	grossOutputsByName := make(map[string]float64)
	grossInputsByName := make(map[string]float64)
	for recipeID, machineCount := range machineByRecipeID {
		if machineCount <= 0 {
			continue
		}
		item, ok := recipeByID[recipeID]
		if !ok {
			continue
		}
		power := item.PowerKW * float64(machineCount)
		totalPowerKW += power
		scale := float64(machineCount)
		for _, output := range item.Outputs {
			grossOutputsByName[output.Name] += perMinute(output.Amount, item.CycleSeconds) * scale
		}
		for _, input := range item.Inputs {
			grossInputsByName[input.Name] += perMinute(input.Amount, item.CycleSeconds) * scale
		}
		recipeRows = append(recipeRows, requirementRecipeRow{
			RecipeID:     recipeID,
			RecipeName:   item.Name,
			DeviceModel:  item.DeviceModel,
			EffectMode:   item.EffectMode,
			MachineCount: machineCount,
			PowerKW:      power,
		})
	}
	sort.Slice(recipeRows, func(i, j int) bool {
		return recipeRows[i].MachineCount > recipeRows[j].MachineCount
	})

	actualOutputs := make([]requirementMaterialAmount, 0, len(grossOutputsByName))
	for name, amount := range grossOutputsByName {
		if amount <= requirementEPS {
			continue
		}
		actualOutputs = append(actualOutputs, requirementMaterialAmount{Name: name, Amount: amount})
	}
	sort.Slice(actualOutputs, func(i, j int) bool { return actualOutputs[i].Amount > actualOutputs[j].Amount })

	actualInputs := make([]requirementMaterialAmount, 0, len(grossInputsByName))
	for name, amount := range grossInputsByName {
		if amount <= requirementEPS {
			continue
		}
		actualInputs = append(actualInputs, requirementMaterialAmount{Name: name, Amount: amount})
	}
	sort.Slice(actualInputs, func(i, j int) bool { return actualInputs[i].Amount > actualInputs[j].Amount })

	externalInputs := make([]requirementMaterialAmount, 0)
	unresolvedCraftables := make([]requirementMaterialAmount, 0)
	for name, need := range requirement {
		if need <= requirementEPS {
			continue
		}
		item := requirementMaterialAmount{Name: name, Amount: need}
		if _, ok := optionsByMaterial[name]; ok {
			unresolvedCraftables = append(unresolvedCraftables, item)
		} else {
			externalInputs = append(externalInputs, item)
		}
	}
	sort.Slice(externalInputs, func(i, j int) bool { return externalInputs[i].Amount > externalInputs[j].Amount })
	sort.Slice(unresolvedCraftables, func(i, j int) bool { return unresolvedCraftables[i].Amount > unresolvedCraftables[j].Amount })
	if len(unresolvedCraftables) > 0 {
		warnings = append(warnings, "部分可制造材料未能被完全反推，请检查是否有循环依赖或多路径配方。")
	}

	totalExternalInputs := 0.0
	for _, item := range externalInputs {
		totalExternalInputs += item.Amount
	}

	return requirementPlanResult{
		RecipeRows:           recipeRows,
		ExternalInputs:       externalInputs,
		UnresolvedCraftables: unresolvedCraftables,
		ActualOutputs:        actualOutputs,
		ActualInputs:         actualInputs,
		TotalPowerKW:         totalPowerKW,
		TotalExternalInputs:  totalExternalInputs,
		Warnings:             warnings,
	}
}
