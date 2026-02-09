package keeper

import (
	"structs/x/structs/types"
)

// attributeCache holds a single attribute value with change tracking.
type StructAttributeCache struct {
    CC              *CurrentContext
    StructAttributeId string
	Value           uint64
	Loaded          bool
	Changed         bool
}


func (cache *StructAttributeCache) IsChanged() bool {
	return cache.Changed
}

func (cache *StructAttributeCache) ID() string {
	return cache.StructAttributeId
}

func (cache *StructAttributeCache) Commit() {
    if cache.Loaded && cache.Changed {
        cache.Changed = false
    	cache.CC.k.logger.Info("Updating Struct Attribute From Cache", "StructAttributeId", cache.StructAttributeId, "value", cache.Value)

        if cache.Deleted {
            cache.CC.k.ClearStructAttribute(cache.CC.ctx, cache.StructAttributeId)
        } else {
            cache.CC.k.SetStructAttribute(cache.CC.ctx, cache.StructAttributeId, cache.Value)
        }
    }
}







