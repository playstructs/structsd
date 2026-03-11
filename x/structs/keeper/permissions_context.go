package keeper

import (
	"structs/x/structs/types"

	"strings"
	"strconv"
)

// PermissionedObject represents any cache within the permission system
type PermissionedObject interface {
	// ID returns the unique identifier for this cache
	ID() string

	// Ownership Details
	GetOwnerId() string
	GetOwner() *PlayerCache

    // Grid stuff
    CanAllocateAsSourceBy(*PlayerCache) error

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


// GetPermissionsGuildRank returns highestRank, caching the result.
func (cc *CurrentContext) GetPermissionsGuildRank(object PermissionedObject, activePlayer *PlayerCache, permissionType types.Permission) uint64 {
    id := GuildRankPermissionID(object.ID(), activePlayer.GetPlayerId(), permissionType)

	if cache, exists := cc.permissionsGuildRank[id]; exists {
		return cc.permissionsGuildRank[id].HighestRank
	}

	highestRank := cc.k.GetHighestGuildRankForPermission(cc.ctx, object.ID(), activePlayer.GetPlayerId(), permissionType)

	cc.permissions[id] = &PermissionsGuildRankCache{
	    CC:                     cc,
	    PermissionGuildRankID:  id,
        ObjectId:               object.ID(),
        PlayerId:               activePlayer.GetPlayerId(),
        Permission:             permissionType,
      	HighestRank:            highestRank,
	    Loaded:                 true,
    }

	return cc.permissions[id].HighestRank
}



// SetPermissionsGuildRank returns highestRank, caching the result.
func (cc *CurrentContext) SetPermissionsGuildRank(object PermissionedObject, activePlayer *PlayerCache, permissionType types.Permission, highestRank uint64) *PermissionsGuildRankCache {
    id := GuildRankPermissionID(object.ID(), activePlayer.GetPlayerId(), permissionType)

	cc.permissions[id] = &PermissionsGuildRankCache{
	    CC:                     cc,
	    PermissionGuildRankID:  id,
        ObjectId:               object.ID(),
        PlayerId:               activePlayer.GetPlayerId(),
        Permission:             permissionType,
      	HighestRank:            highestRank,
	    Loaded:                 true,
	    Changed:                true,
    }

	return cc.permissions[id]
}


func (cc *CurrentContext) GetPermissionedObject(objectId string) PermissionedObject {
	if objectId == "" {
		return nil
	}
	parts := strings.Split(objectId, "-")
	if len(parts) < 2 {
		return nil
	}
	typeNum, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return nil
	}
	switch types.ObjectType(typeNum) {
        case types.ObjectType_guild:
            return cc.GetGuild(objectId)
        case types.ObjectType_player:
            player, err := cc.GetPlayer(objectId)
            if err != nil {
                return nil
            }
            return player
        case types.ObjectType_planet:
            return cc.GetPlanet(objectId)
        case types.ObjectType_reactor:
            return cc.GetReactor(objectId)
        case types.ObjectType_substation:
            return cc.GetSubstation(objectId)
        case types.ObjectType_struct:
            return cc.GetStruct(objectId)
        case types.ObjectType_allocation:
            allocation, found := cc.GetAllocation(objectId)
            if !found {
                return nil
            }
            return allocation
        case types.ObjectType_fleet:
            fleet, err := cc.GetFleetById(objectId)
            if err != nil {
                return nil
            }
            return fleet
        case types.ObjectType_provider:
            return cc.GetProvider(objectId)
        case types.ObjectType_agreement:
            return cc.GetAgreement(objectId)
        default:
            return nil
	}
	return nil
}

func (cc *CurrentContext) PermissionCheck(object PermissionedObject, activePlayer *PlayerCache, permission types.Permission) error {

    // Really shouldn't have got here but let's do a quick check
    if object == nil || activePlayer == nil {
        return types.NewPermissionError("player", "", "object", "", uint64(permission), "administrate")
    }

    // A check with Permissionless should always return an error
    if permission == types.Permissionless {
        return types.NewPermissionError("player", activePlayer.GetPlayerId(), "object", object.ID(), uint64(permission), "administrate")
    }

    // Check the Active Player exists
    if !activePlayer.HasPlayerAccount() {
        return types.NewPlayerRequiredError(activePlayer.GetActiveAddress(), "administrate")
    }

    // Make sure the address calling this has request permissions
    if !cc.PermissionHasAll(activePlayer.GetActiveAddressPermissionID(), permission) {
        return types.NewPermissionError("address", activePlayer.GetActiveAddress(), "", "", uint64(permission), "administrate")
    }

    // If the player is the owner, it's an easy yes
    if object.GetOwnerId() == activePlayer.ID() {
        return nil
    }

    if cc.PermissionHasAll(GetObjectPermissionIDBytes(object.ID(), activePlayer.GetPlayerId()), permission) {
        return nil
    }

    // rank(object / activePlayer.GetGuild() / permission) => activePlayer.GetGuildRank()
    if activePlayer.GetGuildId() != "" {
        if cc.GetHighestGuildRankForPermission(object.ID(), activePlayer.GetGuildId(), permission) >= activePlayer.GetGuildRank() {
            return nil
        }
    }

    return types.NewPermissionError("player", activePlayer.GetPlayerId(), "object", object.ID(), uint64(permission), "administrate")
}

