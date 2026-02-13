package keeper

import (
	"structs/x/structs/types"
)

// GetPermission returns a struct attribute value, caching the result.
func (cc *CurrentContext) GetPermissions(permissionId []byte) types.Permission {
	if cache, exists := cc.permissions[string(permissionId)]; exists {
		return cache.Value
	}

	value := cc.k.GetPermissionsByBytes(cc.ctx, permissionId)
	cc.permissions[string(permissionId)] = &PermissionsCache{
	    CC:     cc,
	    PermissionId: permissionId,
	    Value:  value,
	    Loaded: true,
    }
	return value
}

func (cc *CurrentContext) SetPermissions(permissionId []byte, value types.Permission) {
	cc.permissions[string(permissionId)] = &PermissionsCache{
 	    CC:     cc,
 	    PermissionId: permissionId,
	    Value: value,
	    Loaded: true,
	    Deleted: false,
	    Changed: true,
	}
}

func (cc *CurrentContext) ClearPermissions(permissionId []byte) {
	cc.permissions[string(permissionId)] = &PermissionsCache{
 	    CC:                 cc,
 	    PermissionId:    permissionId,
	    Value: types.Permissionless,
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

