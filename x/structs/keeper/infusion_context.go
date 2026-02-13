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

func (cc *CurrentContext) GetAllInfusionByDestination(destinationId string) (infusions []*InfusionCache) {
    infusionIds := cc.k.GetAllInfusionIdsByDestination(cc.ctx, destinationId)
    for _, infusionId := range infusionIds {
        infusion := cc.GetInfusionById(infusionId)
        infusions = append(infusions, infusion)
    }
    return
}

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
            if (infusion.CheckInfusion() == nil && infusion.GetInfusion().Power == 0 && infusion.GetInfusion().Defusing == 0) {
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