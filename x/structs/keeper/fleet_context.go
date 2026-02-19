package keeper

import (
	"structs/x/structs/types"
    "strings"
    "strconv"
)

// GetFleet returns a FleetCache by ID, loading from store if not already cached.
func (cc *CurrentContext) GetFleetById(fleetId string) (*FleetCache, error) {
    if ! strings.HasPrefix(fleetId, "9-") {
        return &FleetCache{}, types.NewObjectNotFoundError("fleet", fleetId)
    }

    fleetIdDetails := strings.Split(fleetId, "-")
    if len(fleetIdDetails) != 2 {
        return &FleetCache{}, types.NewObjectNotFoundError("fleet", fleetId)
    }

    index, err := strconv.ParseUint(fleetIdDetails[1], 10, 64)
    if err != nil {
        return &FleetCache{}, types.NewObjectNotFoundError("fleet", fleetId)
    }
    return cc.GetFleet(index)
}

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

func (cc *CurrentContext) GenesisImportFleet(fleet types.Fleet, homePlanetId string) {
	fleet.Status = types.FleetStatus_onStation
	fleet.LocationType = types.ObjectType_planet
	fleet.LocationId = homePlanetId
	fleet.LocationListForward = ""
	fleet.LocationListBackward = ""

	cache, _ := cc.GetFleetById(fleet.Id)
	cache.Fleet = fleet
	cache.FleetLoaded = true
	cache.Changed = true

}

