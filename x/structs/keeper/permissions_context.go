package keeper

import (
	"structs/x/structs/types"

	"strconv"
	"strings"
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
}

// GetPermission returns a struct attribute value, caching the result.
func (cc *CurrentContext) GetPermissions(permissionId []byte) types.Permission {
	if cache, exists := cc.permissions[string(permissionId)]; exists {
		return cache.Value
	}

	value := cc.k.GetPermissionsByBytes(cc.ctx, permissionId)
	cc.permissions[string(permissionId)] = &PermissionsCache{
		CC:           cc,
		PermissionId: permissionId,
		Value:        value,
		Loaded:       true,
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
		CC:           cc,
		PermissionId: permissionId,
		Value:        value,
		Loaded:       true,
		Deleted:      false,
		Changed:      true,
	}
}

func (cc *CurrentContext) ClearPermissions(permissionId []byte) {
	cc.permissions[string(permissionId)] = &PermissionsCache{
		CC:           cc,
		PermissionId: permissionId,
		Value:        types.Permissionless,
		Loaded:       true,
		Changed:      true,
		Deleted:      true,
	}
}

func (cc *CurrentContext) ClearPermissionsForObject(objectId string) {
	deletedKeys := cc.k.ClearPermissionByObject(cc.ctx, objectId)

	for _, deleted := range deletedKeys {
		delete(cc.permissions, deleted)
	}

	cc.k.ClearPermissionGuildRankByObject(cc.ctx, objectId)

	prefix := objectId + "/"
	for id := range cc.permissionsGuildRank {
		if strings.HasPrefix(id, prefix) {
			delete(cc.permissionsGuildRank, id)
		}
	}
}

func (cc *CurrentContext) PermissionAdd(permissionId []byte, flag types.Permission) types.Permission {
	currentFlags := cc.GetPermissions(permissionId)
	newFlags := currentFlags | flag
	cc.SetPermissions(permissionId, newFlags)
	return newFlags
}

func (cc *CurrentContext) PermissionRemove(permissionId []byte, flag types.Permission) types.Permission {
	currentFlags := cc.GetPermissions(permissionId)
	newFlags := currentFlags &^ flag
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

// GetPermissionsGuildRank returns (worstAllowedRank, exists). exists is false when no record is stored or after revoke.
func (cc *CurrentContext) GetPermissionsGuildRank(object PermissionedObject, guild *GuildCache, permissionType types.Permission) (uint64, bool) {
	id := GuildRankPermissionID(object.ID(), guild.ID(), permissionType)

	if c, exists := cc.permissionsGuildRank[id]; exists {
		return c.WorstAllowedRank, c.Exists
	}

	worstAllowedRank, ok := cc.k.GetGuildRankForPermission(cc.ctx, object.ID(), guild.ID(), permissionType)

	cc.permissionsGuildRank[id] = &PermissionsGuildRankCache{
		CC:                    cc,
		PermissionGuildRankID: id,
		ObjectId:              object.ID(),
		GuildId:               guild.ID(),
		Permission:            permissionType,
		WorstAllowedRank:      worstAllowedRank,
		Loaded:                true,
		Exists:                ok,
	}

	return worstAllowedRank, ok
}

// SetPermissionsGuildRank caches the guild rank permission for commit.
func (cc *CurrentContext) SetPermissionsGuildRank(object PermissionedObject, guild *GuildCache, permissionType types.Permission, worstAllowedRank uint64) *PermissionsGuildRankCache {
	id := GuildRankPermissionID(object.ID(), guild.ID(), permissionType)

	cc.permissionsGuildRank[id] = &PermissionsGuildRankCache{
		CC:                    cc,
		PermissionGuildRankID: id,
		ObjectId:              object.ID(),
		GuildId:               guild.ID(),
		Permission:            permissionType,
		WorstAllowedRank:      worstAllowedRank,
		Loaded:                true,
		Changed:               true,
		Exists:                true,
	}

	return cc.permissionsGuildRank[id]
}

// RemovePermissionsGuildRank marks the guild rank permission as deleted in the cache; Commit() will call RemoveGuildRankPermission.
func (cc *CurrentContext) RemovePermissionsGuildRank(object PermissionedObject, guild *GuildCache, permissionType types.Permission) {
	id := GuildRankPermissionID(object.ID(), guild.ID(), permissionType)

	cc.permissionsGuildRank[id] = &PermissionsGuildRankCache{
		CC:                    cc,
		PermissionGuildRankID: id,
		ObjectId:              object.ID(),
		GuildId:               guild.ID(),
		Permission:            permissionType,
		Loaded:                true,
		Changed:               true,
		Deleted:               true,
		Exists:                false,
	}
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

	// rank(object / activePlayer.GetGuild() / permission) => activePlayer.GetGuildRank(); only grant if a record exists
	if activePlayer.GetGuildId() != "" {
		worstAllowedRank, exists := cc.GetPermissionsGuildRank(object, activePlayer.GetGuild(), permission)
		if exists && worstAllowedRank > 0 && worstAllowedRank >= activePlayer.GetGuildRank() {
			return nil
		}
	}

	return types.NewPermissionError("player", activePlayer.GetPlayerId(), "object", object.ID(), uint64(permission), "administrate")
}
