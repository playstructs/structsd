package keeper

// SetGeneratorPowerEnabled updates output atomically with the owner/destination
// deltas. Validation precedes any mutation, including on worker error paths.
func (cache *InfusionCache) SetGeneratorPowerEnabled(enabled bool) error {
	if err := cache.CheckInfusion(); err != nil {
		return err
	}
	if err := cache.Infusion.ValidateGeneratorPower(); err != nil {
		return err
	}
	disabled := !enabled
	if cache.Infusion.GeneratorDisabled == disabled {
		return nil
	}
	oldFuel := cache.Infusion.Fuel
	_, oldDestination, oldOwner := cache.Infusion.GetPowerDistribution()
	cache.Infusion.GeneratorDisabled = disabled
	cache.Infusion.Recalculate()
	cache.applyGridDeltas(oldFuel, oldDestination, oldOwner)
	cache.Changed = true
	return nil
}
