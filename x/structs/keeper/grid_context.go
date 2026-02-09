package keeper

import (
    "strings"
    "structs/x/structs/types"
)


// GetGridAttribute returns a grid attribute value, caching the result.
func (cc *CurrentContext) GetGridAttribute(gridAttributeId string) uint64 {
	if cache, exists := cc.gridAttributes[gridAttributeId]; exists {
		return cache.Value
	}

	value := cc.k.GetGridAttribute(cc.ctx, gridAttributeId)
	cc.gridAttributes[gridAttributeId] = &GridAttributeCache{
	    CC:     cc,
	    GridAttributeId: gridAttributeId,
	    Value:  value,
	    Loaded: true,
    }
	return value
}

func (cc *CurrentContext) SetGridAttribute(gridAttributeId string, value uint64) {
	cc.gridAttributes[gridAttributeId] = &GridAttributeCache{
 	    CC:     cc,
 	    GridAttributeId: gridAttributeId,
	    Value: value,
	    Loaded: true,
	    Deleted: false,
	    Changed: true,
	}
}

func (cc *CurrentContext) ClearGridAttribute(gridAttributeId string) {
	cc.gridAttributes[gridAttributeId] = &GridAttributeCache{
 	    CC:                 cc,
 	    GridAttributeId:    gridAttributeId,
	    Value: 0,
	    Loaded: true,
	    Changed: true,
	    Deleted: true,
	}
}

// SetGridAttributeIncrement increments a grid attribute
func (cc *CurrentContext) SetGridAttributeIncrement(gridAttributeId string, delta uint64) uint64 {
	current := cc.GetGridAttribute(gridAttributeId)
	newValue := current + delta
	cc.SetGridAttribute(gridAttributeId, newValue)
	return newValue
}

// SetGridAttributeDecrement decrements a grid attribute
// Will not go below zero.
func (cc *CurrentContext) SetGridAttributeDecrement(gridAttributeId string, delta uint64) uint64 {
	current := cc.GetGridAttribute(gridAttributeId)
	var newValue uint64
	if delta < current {
		newValue = current - delta
	}
	cc.SetGridAttribute(gridAttributeId, newValue)
	return newValue
}

// Updates a Grid Attribute by first removing the old amount and then adding the new amount
func (cc *CurrentContext) SetGridAttributeDelta(gridAttributeId string, oldAmount uint64, newAmount uint64) (uint64) {
	currentAmount := cc.GetGridAttribute(gridAttributeId)

	var resetAmount uint64
	if oldAmount < currentAmount {
		resetAmount = currentAmount - oldAmount
	}
	amount := resetAmount + newAmount

    cc.SetGridAttribute(gridAttributeId, amount)

	return amount
}

func (cc *CurrentContext) UpdateSubstationConnectionCapacity(objectId string) {
    if strings.HasPrefix(objectId, "4-") {

        capacityAttributeId             := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)
        loadAttributeId                 := GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)
        connectionCapacityAttributeId   := GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId)
        connectionCountAttributeId      := GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, objectId)

        capacity    := cc.GetGridAttribute(capacityAttributeId)
        load        := cc.GetGridAttribute(loadAttributeId)

        if capacity > load {
            availableCapacity := capacity - load

            connectionCount := cc.GetGridAttribute(connectionCountAttributeId)
            if connectionCount == 0 {
                connectionCount = 1
            }

            cc.SetGridAttribute(connectionCapacityAttributeId, availableCapacity/connectionCount)
        } else {
            cc.SetGridAttribute(connectionCapacityAttributeId, 0)
        }
    }
}



