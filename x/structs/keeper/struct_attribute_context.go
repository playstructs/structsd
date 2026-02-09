package keeper

// GetStructAttribute returns a struct attribute value, caching the result.
func (cc *CurrentContext) GetStructAttribute(structAttributeId string) uint64 {
	if cache, exists := cc.structAttributes[structAttributeId]; exists {
		return cache.Value
	}

	value := cc.k.GetStructAttribute(cc.ctx, structAttributeId)
	cc.structAttributes[structAttributeId] = &StructAttributeCache{
	    CC:     cc,
	    StructAttributeId: structAttributeId,
	    Value:  value,
	    Loaded: true,
    }
	return value
}

func (cc *CurrentContext) SetStructAttribute(structAttributeId string, value uint64) {
	cc.structAttributes[structAttributeId] = &StructAttributeCache{
 	    CC:     cc,
 	    StructAttributeId: structAttributeId,
	    Value: value,
	    Loaded: true,
	    Deleted: false,
	    Changed: true,
	}
}

func (cc *CurrentContext) ClearStructAttribute(structAttributeId string) {
	cc.structAttributes[structAttributeId] = &StructAttributeCache{
 	    CC:                 cc,
 	    StructAttributeId:    structAttributeId,
	    Value: 0,
	    Loaded: true,
	    Changed: true,
	    Deleted: true,
	}
}

// SetStructAttributeIncrement increments a struct attribute
func (cc *CurrentContext) SetStructAttributeIncrement(attributeId string, delta uint64) uint64 {
	current := cc.GetStructAttribute(attributeId)
	newValue := current + delta

	cc.k.logger.Info("Struct Change (Increment)", "structAttributeId", structAttributeId, "incrementAmount", incrementAmount)
	cc.SetStructAttribute(attributeId, newValue)
	return newValue
}

// SetStructAttributeDecrement decrements a struct attribute
// Will not go below zero.
func (cc *CurrentContext) SetStructAttributeDecrement(attributeId string, delta uint64) uint64 {
	current := cc.GetStructAttribute(attributeId)
	var newValue uint64
	if delta < current {
		newValue = current - delta
	}

	cc.k.logger.Info("Struct Change (Decrement)", "structAttributeId", structAttributeId, "decrementAmount", decrementAmount)
	cc.SetStructAttribute(attributeId, newValue)
	return newValue
}

// Updates a Struct Attribute by first removing the old amount and then adding the new amount
func (cc *CurrentContext) SetStructAttributeDelta(structAttributeId string, oldAmount uint64, newAmount uint64) (uint64) {
	currentAmount := cc.GetStructAttribute(structAttributeId)

	var resetAmount uint64
	if oldAmount < currentAmount {
		resetAmount = currentAmount - oldAmount
	}
	amount := resetAmount + newAmount

    cc.k.logger.Info("Struct Change (Delta)", "structAttributeId", structAttributeId, "oldAmount", oldAmount, "newAmount", newAmount)
    cc.SetStructAttribute(attributeId, amount)

	return amount
}

/* The Struct Attribute Store also supports bitwise flags */

func (cc *CurrentContext) SetStructAttributeFlagAdd(structAttributeId string, flag uint64) uint64 {
    currentFlags    := cc.GetStructAttribute(structAttributeId)
    newFlags        := currentFlags | flag
    cc.SetStructAttribute(structAttributeId, newFlags)
	return newFlags
}

func (cc *CurrentContext) SetStructAttributeFlagRemove(structAttributeId string, flag uint64) uint64 {
    currentFlags    := cc.GetStructAttribute(structAttributeId)
    newFlags        := currentFlags &^ flag
    cc.SetStructAttribute(structAttributeId, newFlags)
	return newFlags
}

func (cc *CurrentContext) StructAttributeFlagHasAll(structAttributeId string, flag uint64) bool {
    currentFlags := cc.GetStructAttribute(structAttributeId)
	return currentFlags&flag == flag
}

func (cc *CurrentContext) StructAttributeFlagHasOneOf(structAttributeId string, flag uint64) bool {
    currentFlags := cc.GetStructAttribute(structAttributeId)
	return currentFlags&flag != 0
}