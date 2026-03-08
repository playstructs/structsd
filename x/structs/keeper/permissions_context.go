package keeper

import (
	"structs/x/structs/types"
)

// PermissionedObject represents any cache within the permission system
type PermissionedObject interface {
	// ID returns the unique identifier for this cache
	ID() string

	// Ownership Details
	GetOwnerId() string
	GetOwner() PlayerCache

    // Rank Details

}

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

func (cc *CurrentContext) GenesisImportPermission(permissionId []byte, value types.Permission) {
	cc.permissions[string(permissionId)] = &PermissionsCache{
		CC:           cc,
		PermissionId: permissionId,
		Value:        value,
		Loaded:       true,
		Changed:      true,
	}
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



func (cache *CurrentContext) PermissionCheck(object *PermissionedObject, activePlayer *PlayerCache, permission types.Permission) (error) {

    // Make sure the address calling this has request permissions
    if (!cache.PermissionHasAll(activePlayer.GetActiveAddressPermissionID(), permission)) {
        return types.NewPermissionError("address", activePlayer.GetActiveAddress(), "", "", uint64(permission), "administrate")
    }

    // If the player isn't the owner, check deeper
    if (object.GetOwnerId() != activePlayer.ID()) {
        if (!cache.PermissionHasAll(GetObjectPermissionIDBytes(object.ID(), activePlayer.GetPlayerId()), permission)) {
           return types.NewPermissionError("player", activePlayer.GetPlayerId(), "object", object.ID(), uint64(permission), "administrate")
        }
    }

    return nil
}
