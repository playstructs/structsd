package keeper

import (
	"structs/x/structs/types"
)

// attributeCache holds a single attribute value with change tracking.
type GridAttributeCache struct {
    CC              *CurrentContext
    GridAttributeId string
	Value           uint64
	Loaded          bool
	Changed         bool
	Deleted         bool

    // Maybe, Maybe Not.
    ObjectId        string
    AttributeType   types.GridAttributeType
}


func (cache *GridAttributeCache) IsChanged() bool {
	return cache.Changed
}

func (cache *GridAttributeCache) ID() string {
	return cache.GridAttributeId
}

func (cache *GridAttributeCache) Commit() {
    if cache.Loaded && cache.Changed {
        cache.Changed = false
    	cache.CC.k.logger.Info("Updating Grid Attribute From Cache", "GridAttributeId", cache.GridAttributeId, "value", cache.Value)

        if cache.Deleted {
            cache.CC.k.ClearGridAttribute(cache.CC.ctx, cache.GridAttributeId)
        } else {
            cache.CC.k.SetGridAttribute(cache.CC.ctx, cache.GridAttributeId, cache.Value)
        }
    }
}

func (cache *GridAttributeCache) Clear() {
    cache.Value = 0
    cache.Changed = true
    cache.Deleted = true
}
