package keeper

import (
    "structs/x/structs/types"
    "strings"
)


func (cc *CurrentContext) GetInfusionById(infusionKey string) (*InfusionCache, bool) {
    if cache, exists := cc.infusions[infusionKey]; exists {
        return cache, exists
    }

    return &InfusionCache{}, false
}


func (cc *CurrentContext) GetInfusion(destinationType types.ObjectType, destinationId string, address string) *InfusionCache {
    infusionKey := destinationId + "/" + address

    if cache, exists := cc.infusions[infusionKey]; exists {
        return cache
    }

    cc.infusions[infusionKey] = &InfusionCache{
        DestinationType:                destinationType,
        DestinationId:                  destinationId,
        Address:                        address,
        CC:                             cc,
        DestinationFuelAttributeId:     GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, destinationId),
        DestinationCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
    }

    return cc.infusions[infusionKey]
}

func (cc *CurrentContext) ProcessInfusionDestructionQueue() {
    for {
        queue := cc.k.GetInfusionDestructionQueue(cc.ctx, true)
        if len(queue) == 0 {
            break
        }

        for _, infusionId := range queue {
            // infusionId format: "destinationId-address" (e.g. "3-1-cosmos1abc...")
            // destinationId itself contains "-", so split on the last dash
            lastDash := strings.LastIndex(infusionId, "-")
            if lastDash == -1 {
                continue
            }
            destinationId := infusionId[:lastDash]
            address := infusionId[lastDash+1:]

            infusion, found := cc.k.GetInfusion(cc.ctx, destinationId, address)
            if found && infusion.Power == 0 && infusion.Defusing == 0 {
                cache := cc.GetInfusion(infusion.DestinationType, destinationId, address)
                cache.Destroy()
            }
        }
    }
}

func (cc *CurrentContext) DestroyAllInfusions(infusionIds []string) {
	for _, infusionId := range infusionIds {
		infusion, found := cc.GetInfusionById(infusionId)
		if found {
		    infusion.Destroy()
		}
	}
}