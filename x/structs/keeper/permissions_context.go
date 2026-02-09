package keeper

import (
	"structs/x/structs/types"
)

// GetPermission returns a struct attribute value, caching the result.
func (cc *CurrentContext) GetPermissions(permissionId []byte) uint64 {
	if cache, exists := cc.permissions[permissionId]; exists {
		return cache.Value
	}

	value := cc.k.GetPermission(cc.ctx, permissionId)
	cc.permissions[permissionId] = &PermissionsCache{
	    CC:     cc,
	    PermissionId: permissionId,
	    Value:  value,
	    Loaded: true,
    }
	return value
}

func (cc *CurrentContext) SetPermissions(permissionId []byte, value uint64) {
	cc.permissions[permissionId] = &PermissionsCache{
 	    CC:     cc,
 	    PermissionId: permissionId,
	    Value: value,
	    Loaded: true,
	    Deleted: false,
	    Changed: true,
	}
}

func (cc *CurrentContext) ClearPermissions(permissionId []byte) {
	cc.permissions[permissionId] = &PermissionsCache{
 	    CC:                 cc,
 	    PermissionId:    permissionId,
	    Value: 0,
	    Loaded: true,
	    Changed: true,
	    Deleted: true,
	}
}


func (cc *CurrentContext) PermissionAdd(permissionId []byte, flag types.Permission) types.Permission {
    currentFlags    := cc.GetPermissions(permissionId)
    newFlags        := currentFlags | flag
    cc.SetPermissions(permissionId, newFlags)
	return newFlags
}

func (cc *CurrentContext) PermissionRemove(permissionId []byte, flag types.Permission) types.Permission {
    currentFlags    := cc.GetPermissions(permissionId)
    newFlags        := currentFlags &^ flag
    cc.SetPermissions(permissionId, newFlags)
	return newFlags
}

func (cc *CurrentContext) PermissionHasAll(permissionId []byte, flag types.Permission) bool {
    currentFlags := cc.GetPermissions(permissionId)
	return currentFlags&flag == flag
}

func (cc *CurrentContext) PermissionHasOneOf(permissionId []byte, flag types.Permission) bool {
    currentFlags := cc.GetPermissions(permissionId)
	return currentFlags&flag != 0
}

