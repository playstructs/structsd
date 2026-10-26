package keeper

import (
	"structs/x/structs/types"
)

/* GetFleetById resolves a fleet id, or returns a detached cache and an error.
 *
 * The detached cache is the trap: it is not in cc.fleets, so marking it Changed
 * does nothing at CommitAll, and its CC is nil, so calling a method on it
 * panics. Callers must check the error - discarding it turns a malformed id into
 * a silent no-op or a panic, depending only on what the caller does next.
 *
 * The id rule itself lives in types.ParseFleetId, because GenesisState.Validate
 * has to apply the same one and cannot import this package.
 */
func (cc *CurrentContext) GetFleetById(fleetId string) (*FleetCache, error) {
    index, err := types.ParseFleetId(fleetId)
    if err != nil {
        return &FleetCache{}, err
    }
    return cc.GetFleet(index)
}

// GetFleet returns a FleetCache by index, loading from store if not already cached.
func (cc *CurrentContext) GetFleet(index uint64) (*FleetCache, error) {
	if cache, exists := cc.fleets[index]; exists {
		return cache, nil
	}

	cc.fleets[index] = &FleetCache{
                              FleetId: GetObjectID(types.ObjectType_fleet, index),
                              PlayerId: GetObjectID(types.ObjectType_player, index),
                              Index: index,
                              CC: cc,
                              Changed: false,
                          }

	return cc.fleets[index], nil
}

/* GenesisImportFleet imports one fleet, or reports that its id is unusable.
 *
 * The error return is the fix rather than a nicety. GetFleetById hands back a
 * cache that is not in cc.fleets when the id is malformed, and CommitAll only
 * commits what is in that map - so discarding the error dropped the fleet in
 * total silence while the players and structs pointing at it imported normally,
 * leaving those owners with fleet-bound assets and no fleet. Nothing downstream
 * noticed: GenesisState.Validate did not look at FleetList, and the caller had
 * no error to check.
 */
func (cc *CurrentContext) GenesisImportFleet(fleet types.Fleet, homePlanetId string) error {
	fleet.Status = types.FleetStatus_onStation
	fleet.LocationType = types.ObjectType_planet
	fleet.LocationId = homePlanetId
	fleet.LocationListForward = ""
	fleet.LocationListBackward = ""

	cache, err := cc.GetFleetById(fleet.Id)
	if err != nil {
		return err
	}

	cache.Fleet = fleet
	cache.FleetLoaded = true
	cache.Changed = true

	return nil
}

