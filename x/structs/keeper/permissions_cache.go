package keeper

import (
	"structs/x/structs/types"
)

// attributeCache holds a single attribute value with change tracking.
type PermissionsCache struct {
    CC              *CurrentContext
    PermissionId    []byte
	Value           types.Permission
	Loaded          bool
	Changed         bool
	Deleted         bool
}


func (cache *PermissionsCache) IsChanged() bool {
	return cache.Changed
}

func (cache *PermissionsCache) ID() []byte {
	return cache.PermissionId
}

func (cache *PermissionsCache) Commit() {
    if cache.Loaded && cache.Changed {
        cache.Changed = false
    	cache.CC.k.logger.Info("Updating Permission From Cache", "PermissionsId", cache.PermissionId, "value", cache.Value)

        if cache.Deleted {
            cache.CC.k.PermissionClearAll(cache.CC.ctx, cache.PermissionId)
        } else {
            cache.CC.k.SetPermissionsByBytes(cache.CC.ctx, cache.PermissionId, cache.Value)
        }
    }
}







