package antiban

// WarmupBudget calculates the effective daily budget for an instance
// based on how many days it has been warming up.
// Schedule:
//   Days  1-3:  10% of full budget
//   Days  4-7:  25%
//   Days  8-10: 50%
//   Days 11-14: 75%
//   Days  15+:  100%
func WarmupBudget(fullBudget int, warmupDay int) int {
	if warmupDay <= 0 {
		return fullBudget
	}

	var percentage float64
	switch {
	case warmupDay <= 3:
		percentage = 0.10
	case warmupDay <= 7:
		percentage = 0.25
	case warmupDay <= 10:
		percentage = 0.50
	case warmupDay <= 14:
		percentage = 0.75
	default:
		return fullBudget
	}

	budget := int(float64(fullBudget) * percentage)
	if budget < 1 {
		budget = 1
	}
	return budget
}

// WarmupHourlyBudget calculates effective hourly budget during warmup.
func WarmupHourlyBudget(fullBudget int, warmupDay int) int {
	return WarmupBudget(fullBudget, warmupDay)
}

// IsWarmingUp returns true if the instance is still in warmup period.
func IsWarmingUp(warmupDay int) bool {
	return warmupDay > 0 && warmupDay <= 14
}
