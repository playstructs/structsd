package keeper

import (
    "structs/x/structs/types"
    "strings"
    "cosmossdk.io/math"
)


func (cc *CurrentContext) GetInfusionById(infusionKey string) *InfusionCache {

    if cache, exists := cc.infusions[infusionKey]; exists {
        return cache
    }

	infusionIdSplit := strings.Split(infusionKey, "-")
	if len(infusionIdSplit) != 3 {
		return &InfusionCache{}
	}

    destinationId := infusionIdSplit[0] + "-" + infusionIdSplit[1]
    address := infusionIdSplit[2]

    cc.infusions[infusionKey] = &InfusionCache{
        InfusionId:                     infusionKey,
        DestinationId:                  destinationId,
        Address:                        address,
        CC:                             cc,
        DestinationFuelAttributeId:     GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, destinationId),
        DestinationCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
    }

    return cc.infusions[infusionKey]
}


func (cc *CurrentContext) GetInfusion(destinationId string, address string) *InfusionCache {
    infusionKey := destinationId + "-" + address

    if cache, exists := cc.infusions[infusionKey]; exists {
        return cache
    }

    cc.infusions[infusionKey] = &InfusionCache{
        InfusionId:                     infusionKey,
        DestinationId:                  destinationId,
        Address:                        address,
        CC:                             cc,
        DestinationFuelAttributeId:     GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, destinationId),
        DestinationCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
    }

    return cc.infusions[infusionKey]
}

func (cc *CurrentContext) GenesisImportInfusion(infusion types.Infusion) {
	if infusion.DestinationType == types.ObjectType_reactor {
		return
	}
	if infusion.Power == 0 {
		return
	}

	cache := cc.GetInfusion(infusion.DestinationId, infusion.Address)
	cache.Infusion = infusion
	cache.InfusionLoaded = true
	cache.Changed = true

	cache.Infusion.Recalculate()
	cache.applyGridDeltas(0, 0, 0)
}

func (cc *CurrentContext) GetAllInfusionByDestination(destinationId string) (infusions []*InfusionCache) {
    infusionIds := cc.k.GetAllInfusionIdsByDestination(cc.ctx, destinationId)
    for _, infusionId := range infusionIds {
        infusion := cc.GetInfusionById(infusionId)
        infusions = append(infusions, infusion)
    }
    return
}

// UpsertInfusion returns the infusion for a (destination, address) pair,
// creating it if it does not exist yet.
//
// Every caller resolves playerId from the address itself, so it is always that
// address's current owner. An existing record is therefore re-homed rather than
// left alone: the address may have been revoked and re-registered to somebody
// else since the record was written, and only PlayerId can drift that way.
// DestinationId and Address are the cache key, and DestinationType is implied by
// the destinationId prefix, so none of those three can disagree without corrupt
// state.
func (cc *CurrentContext) UpsertInfusion(destinationType types.ObjectType, destinationId string, address string, playerId string) (*InfusionCache){
    infusion := cc.GetInfusion(destinationId, address)

    if infusion.CheckInfusion() != nil {
        infusion.Infusion = types.Infusion{
             DestinationId:      destinationId,
             DestinationType:    destinationType,
             Address:            address,
             PlayerId:           playerId,
             Commission:         math.LegacyZeroDec(),
         }

         infusion.InfusionLoaded = true
         infusion.Changed = true
    } else if infusion.GetInfusion().PlayerId != playerId {
        infusion.SetPlayerId(playerId)
    }
    return infusion
}


func (cc *CurrentContext) ProcessInfusionDestructionQueue() {
    for {
        queue := cc.k.GetInfusionDestructionQueue(cc.ctx, true)
        if len(queue) == 0 {
            break
        }

        for _, infusionId := range queue {
            infusion := cc.GetInfusionById(infusionId)
            if (infusion.CheckInfusion() == nil && infusion.IsEmpty()) {
                infusion.Destroy()
            }
        }
    }
}

func (cc *CurrentContext) DestroyAllInfusions(infusionIds []string) {
	for _, infusionId := range infusionIds {
		infusion := cc.GetInfusionById(infusionId)
		if infusion.CheckInfusion() == nil {
		    infusion.Destroy()
		}
	}
}