package keeper


// attributeCache holds a single attribute value with change tracking.
type PlanetAttributeCache struct {
    CC                  *CurrentContext
    PlanetAttributeId   string
	Value               uint64
	Loaded              bool
	Changed             bool
	Deleted             bool
}


func (cache *PlanetAttributeCache) IsChanged() bool {
	return cache.Changed
}

func (cache *PlanetAttributeCache) ID() string {
	return cache.PlanetAttributeId
}

func (cache *PlanetAttributeCache) Commit() {
    if cache.Loaded && cache.Changed {
        cache.Changed = false
    	cache.CC.k.logger.Info("Updating Planet Attribute From Cache", "PlanetAttributeId", cache.PlanetAttributeId, "value", cache.Value)

        if cache.Deleted {
            cache.CC.k.ClearPlanetAttribute(cache.CC.ctx, cache.PlanetAttributeId)
        } else {
            cache.CC.k.SetPlanetAttribute(cache.CC.ctx, cache.PlanetAttributeId, cache.Value)
        }
    }
}
