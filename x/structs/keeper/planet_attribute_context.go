package keeper

// GetPlanetAttribute returns a planet attribute value, caching the result.
func (cc *CurrentContext) GetPlanetAttribute(planetAttributeId string) uint64 {
	if cache, exists := cc.planetAttributes[planetAttributeId]; exists {
		return cache.Value
	}

	value := cc.k.GetPlanetAttribute(cc.ctx, planetAttributeId)
	cc.planetAttributes[planetAttributeId] = &PlanetAttributeCache{
	    CC:     cc,
	    PlanetAttributeId: planetAttributeId,
	    Value:  value,
	    Loaded: true,
    }
	return value
}

func (cc *CurrentContext) SetPlanetAttribute(planetAttributeId string, value uint64) {
	cc.planetAttributes[planetAttributeId] = &PlanetAttributeCache{
 	    CC:     cc,
 	    PlanetAttributeId: planetAttributeId,
	    Value: value,
	    Loaded: true,
	    Deleted: false,
	    Changed: true,
	}
}

func (cc *CurrentContext) ClearPlanetAttribute(planetAttributeId string) {
	cc.planetAttributes[planetAttributeId] = &PlanetAttributeCache{
 	    CC:                 cc,
 	    PlanetAttributeId:    planetAttributeId,
	    Value: 0,
	    Loaded: true,
	    Changed: true,
	    Deleted: true,
	}
}

// SetPlanetAttributeIncrement increments a planet attribute
func (cc *CurrentContext) SetPlanetAttributeIncrement(attributeId string, delta uint64) uint64 {
	current := cc.GetPlanetAttribute(attributeId)
	newValue := current + delta

	cc.k.logger.Info("Planet Change (Increment)", "planetAttributeId", planetAttributeId, "incrementAmount", incrementAmount)
	cc.SetPlanetAttribute(attributeId, newValue)
	return newValue
}

// SetPlanetAttributeDecrement decrements a planet attribute
// Will not go below zero.
func (cc *CurrentContext) SetPlanetAttributeDecrement(attributeId string, delta uint64) uint64 {
	current := cc.GetPlanetAttribute(attributeId)
	var newValue uint64
	if delta < current {
		newValue = current - delta
	}

	cc.k.logger.Info("Planet Change (Decrement)", "planetAttributeId", planetAttributeId, "decrementAmount", decrementAmount)
	cc.SetPlanetAttribute(attributeId, newValue)
	return newValue
}

// Updates a Planet Attribute by first removing the old amount and then adding the new amount
func (cc *CurrentContext) SetPlanetAttributeDelta(planetAttributeId string, oldAmount uint64, newAmount uint64) (uint64) {
	currentAmount := cc.GetPlanetAttribute(planetAttributeId)

	var resetAmount uint64
	if oldAmount < currentAmount {
		resetAmount = currentAmount - oldAmount
	}
	amount := resetAmount + newAmount

    cc.k.logger.Info("Planet Change (Delta)", "planetAttributeId", planetAttributeId, "oldAmount", oldAmount, "newAmount", newAmount)
    cc.SetPlanetAttribute(attributeId, amount)

	return amount
}

/* The Planet Attribute Store also supports bitwise flags */

func (cc *CurrentContext) SetPlanetAttributeFlagAdd(planetAttributeId string, flag uint64) uint64 {
    currentFlags    := cc.GetPlanetAttribute(planetAttributeId)
    newFlags        := currentFlags | flag
    cc.SetPlanetAttribute(planetAttributeId, newFlags)
	return newFlags
}

func (cc *CurrentContext) SetPlanetAttributeFlagRemove(planetAttributeId string, flag uint64) uint64 {
    currentFlags    := cc.GetPlanetAttribute(planetAttributeId)
    newFlags        := currentFlags &^ flag
    cc.SetPlanetAttribute(planetAttributeId, newFlags)
	return newFlags
}

func (cc *CurrentContext) PlanetAttributeFlagHasAll(planetAttributeId string, flag uint64) bool {
    currentFlags := cc.GetPlanetAttribute(planetAttributeId)
	return currentFlags&flag == flag
}

func (cc *CurrentContext) PlanetAttributeFlagHasOneOf(planetAttributeId string, flag uint64) bool {
    currentFlags := cc.GetPlanetAttribute(planetAttributeId)
	return currentFlags&flag != 0
}