package keeper

import (
	"structs/x/structs/types"
)

// attributeCache holds a single attribute value with change tracking.
type PermissionsCache struct {
    CC              *CurrentContext
    PermissionsId   []byte
	Value           types.Permission
	Loaded          bool
	Changed         bool
}


func (cache *PermissionsCache) IsChanged() bool {
	return cache.Changed
}

func (cache *PermissionsCache) ID() string {
	return cache.PermissionsId
}

func (cache *PermissionsCache) Commit() {
    if cache.Loaded && cache.Changed {
        cache.Changed = false
    	cache.CC.k.logger.Info("Updating Permission From Cache", "PermissionsId", cache.PermissionsId, "value", cache.Value)

        if cache.Deleted {
            cache.CC.k.ClearPermissions(cache.CC.ctx, cache.PermissionsId)
        } else {
            cache.CC.k.SetPermissions(cache.CC.ctx, cache.PermissionsId, cache.Value)
        }
    }
}







