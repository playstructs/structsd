package keeper

import (
	"structs/x/structs/types"
)


// GetPlanet returns a PlanetCache by ID, loading from store if not already cached.
func (cc *CurrentContext) GetPlanet(planetId string) *PlanetCache {
	if cache, exists := cc.planets[planetId]; exists {
		return cache
	}

	cc.planets[planetId] = &PlanetCache{
                PlanetId: planetId,
                CC: cc,

                Changed: false,

                BlockStartRaidAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, planetId),
                BuriedOreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, planetId),

                PlanetaryShieldAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, planetId),
                RepairNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_repairNetworkQuantity, planetId),
                DefensiveCannonQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_defensiveCannonQuantity, planetId),

                CoordinatedGlobalShieldNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_coordinatedGlobalShieldNetworkQuantity, planetId),

                LowOrbitBallisticsInterceptorNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity, planetId),
                AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedLowOrbitBallisticsInterceptorNetworkQuantity, planetId),

                LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator, planetId),
                LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator, planetId),


                OrbitalJammingStationQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_orbitalJammingStationQuantity, planetId),
                AdvancedOrbitalJammingStationQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedOrbitalJammingStationQuantity, planetId),
            }


	return cc.planets[planetId]
}

func (cc *CurrentContext) NewPlanet(creator string, playerId string) (*PlanetCache) {
    planet := types.CreateEmptyPlanet()

	// Create the planet
	count := cc.k.GetPlanetCount(cc.ctx)

	// Set the ID of the appended value
	planetId := GetObjectID(types.ObjectType_planet, count)

	planet.Id = planetId
	planet.SetCreator(creator)
	planet.SetOwner(playerId)

	// Update planet count
	cc.k.SetPlanetCount(cc.ctx, count+1)


    cc.planets[planetId] = &PlanetCache{
                PlanetId: planetId,
                CC: cc,

                Planet: planet,
                PlanetLoaded: true,
                Changed: true,

                BlockStartRaidAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, planetId),
                BuriedOreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, planetId),

                PlanetaryShieldAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, planetId),
                RepairNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_repairNetworkQuantity, planetId),
                DefensiveCannonQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_defensiveCannonQuantity, planetId),

                CoordinatedGlobalShieldNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_coordinatedGlobalShieldNetworkQuantity, planetId),

                LowOrbitBallisticsInterceptorNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity, planetId),
                AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedLowOrbitBallisticsInterceptorNetworkQuantity, planetId),

                LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator, planetId),
                LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator, planetId),


                OrbitalJammingStationQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_orbitalJammingStationQuantity, planetId),
                AdvancedOrbitalJammingStationQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedOrbitalJammingStationQuantity, planetId),
            }


    cc.SetGridAttribute(cc.planets[planetId].BuriedOreAttributeId, types.PlanetStartingOre)
    cc.SetPlanetAttribute(cc.planets[planetId].PlanetaryShieldAttributeId, types.PlanetaryShieldBase)

	return cc.planets[planetId]
}