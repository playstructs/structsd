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

	pfx := objectId + "/"
	for id := range cc.guildRankRegisters {
		if strings.HasPrefix(id, pfx) {
			delete(cc.guildRankRegisters, id)
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

func (cc *CurrentContext) getGuildRankRegister(objectId string, guildId string) *GuildRankRegisterCache {
	pairKey := objectId + "/" + guildId
	if reg, ok := cc.guildRankRegisters[pairKey]; ok {
		return reg
	}
	r := cc.k.ReadGuildRankRegister(cc.ctx, objectId, guildId)
	reg := &GuildRankRegisterCache{
		CC: cc, ObjectId: objectId, GuildId: guildId,
		Original: r, Register: r, Loaded: true,
	}
	cc.guildRankRegisters[pairKey] = reg
	return reg
}

// GetPermissionsGuildRank returns (worstAllowedRank, exists).
// For combined masks, ALL requested bits must have a record; the most restrictive rank is returned.
func (cc *CurrentContext) GetPermissionsGuildRank(object PermissionedObject, guild *GuildCache, permissionType types.Permission) (uint64, bool) {
	reg := cc.getGuildRankRegister(object.ID(), guild.ID())

	var worstRank uint64
	found := false
	for bit := 0; bit < types.PermissionBitCount; bit++ {
		if uint64(permissionType)&(1<<bit) != 0 {
			slotRank := reg.Register[bit]
			if slotRank == 0 {
				return 0, false
			}
			if !found || slotRank < worstRank {
				worstRank = slotRank
			}
			found = true
		}
	}
	return worstRank, found
}

// SetPermissionsGuildRank decomposes the permission mask and sets each bit's rank in the register cache.
func (cc *CurrentContext) SetPermissionsGuildRank(object PermissionedObject, guild *GuildCache, permissionType types.Permission, worstAllowedRank uint64) {
	reg := cc.getGuildRankRegister(object.ID(), guild.ID())
	for bit := 0; bit < types.PermissionBitCount; bit++ {
		if uint64(permissionType)&(1<<bit) != 0 {
			reg.Register[bit] = worstAllowedRank
		}
	}
	reg.Changed = true
}

// RemovePermissionsGuildRank zeros the requested permission bits in the register cache.
func (cc *CurrentContext) RemovePermissionsGuildRank(object PermissionedObject, guild *GuildCache, permissionType types.Permission) {
	reg := cc.getGuildRankRegister(object.ID(), guild.ID())
	for bit := 0; bit < types.PermissionBitCount; bit++ {
		if uint64(permissionType)&(1<<bit) != 0 {
			reg.Register[bit] = 0
		}
	}
	reg.Changed = true
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
		return cc.GetPlayer(objectId)
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

/* requirePrimaryAddressKeepsFullAccess refuses a permission write that would
 * leave a player's current primary address holding less than PermAll.
 *
 * The primary address is the recovery address, and every route out of a
 * downgrade runs back through it: PlayerUpdatePrimaryAddress needs PermAll to
 * rotate to another key, and a grant can only hand over bits the granting
 * address itself holds, so a primary below PermAll cannot restore itself and
 * cannot authorize anything else to. The state is unrecoverable rather than
 * merely reduced, and reaching it takes nothing more than the player's own
 * PermAll key acting on itself - the authorization check passes precisely
 * because they still have the rights they are about to destroy.
 *
 * This blocks nothing anyone wants. A player who wants a narrower everyday key
 * rotates the primary to the address they intend to keep whole first, then
 * downgrades the old one, which this leaves alone.
 */
func requirePrimaryAddressKeepsFullAccess(targetPlayer *PlayerCache, address string, resulting types.Permission) error {
	if address != targetPlayer.GetPrimaryAddress() {
		return nil
	}

	if resulting&types.PermAll == types.PermAll {
		return nil
	}

	return types.NewAddressValidationError(address, "primary_full_access_required")
}

/* SignerPermissionCheck is PermissionCheck's Layer 1 standing alone: the key
 * that signed must itself hold the permission, with no question of what standing
 * its player has on any object.
 *
 * It exists for the policy tiers that deliberately waive the object-level grant.
 * A guild at bypass level `member` says any member may act without holding a
 * grant on the guild - but waiving the grant must not also waive the ceiling.
 * The two are orthogonal: one is about a player's standing in the guild, the
 * other about how far a particular key of that player may reach. Coupling them
 * meant relaxing a guild's join policy silently un-scoped every delegated key
 * its members hold, and the `permissioned` tier next door - which goes through
 * PermissionCheck - kept the ceiling, so the same guild got two different
 * answers about the same key depending on an unrelated setting.
 */
func (cc *CurrentContext) SignerPermissionCheck(permission types.Permission) error {
	wanted := types.PermissionName(permission)

	if permission == types.Permissionless {
		return types.NewPermissionError("address", cc.signerAddress, "", "", uint64(permission), wanted)
	}

	// An empty signer means no address was ever authenticated for this
	// operation, so there is nothing to check against and the only safe answer
	// is no. Same reasoning as PermissionCheck.
	if cc.signerAddress == "" {
		return types.NewPermissionError("address", "", "", "", uint64(permission), wanted)
	}

	if !cc.PermissionHasAll(GetAddressPermissionIDBytes(cc.signerAddress), permission) {
		return types.NewPermissionError("address", cc.signerAddress, "", "", uint64(permission), wanted)
	}

	return nil
}

func (cc *CurrentContext) PermissionCheck(object PermissionedObject, activePlayer *PlayerCache, permission types.Permission) error {

	// The Action carries the bit that was actually wanted rather than a fixed
	// "administrate". Every denial below used to name administrate whatever it was
	// checking, which made unrelated refusals indistinguishable in a log and sent
	// readers looking for the wrong grant.
	wanted := types.PermissionName(permission)

	// Really shouldn't have got here but let's do a quick check
	if object == nil || activePlayer == nil {
		return types.NewPermissionError("player", "", "object", "", uint64(permission), wanted)
	}

	// A check with Permissionless should always return an error
	if permission == types.Permissionless {
		return types.NewPermissionError("player", activePlayer.GetPlayerId(), "object", object.ID(), uint64(permission), wanted)
	}

	// Check the Active Player exists
	if !activePlayer.HasPlayerAccount() {
		return types.NewPlayerRequiredError(cc.signerAddress, wanted)
	}

	// Layer 1: the key that signed must itself hold the permission, whatever
	// standing its player has on the object. An empty signer means no address
	// was ever authenticated for this operation, so there is nothing to check
	// against and the only safe answer is no.
	if cc.signerAddress == "" {
		return types.NewPermissionError("address", "", "object", object.ID(), uint64(permission), wanted)
	}

	if !cc.PermissionHasAll(GetAddressPermissionIDBytes(cc.signerAddress), permission) {
		return types.NewPermissionError("address", cc.signerAddress, "", "", uint64(permission), wanted)
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

	return types.NewPermissionError("player", activePlayer.GetPlayerId(), "object", object.ID(), uint64(permission), wanted)
}

func (cc *CurrentContext) UGCPermissionCheck(object PermissionedObject, activePlayer *PlayerCache) error {
	if err := cc.PermissionCheck(object, activePlayer, types.PermUpdate); err == nil {
		return nil
	}

	owner := object.GetOwner()
	if owner == nil {
		return types.NewPermissionError("player", activePlayer.GetPlayerId(), "object", object.ID(), uint64(types.PermGuildUGCUpdate), "ugc update")
	}

	ownerGuildId := owner.GetGuildId()
	if ownerGuildId == "" {
		return types.NewPermissionError("player", activePlayer.GetPlayerId(), "object", object.ID(), uint64(types.PermGuildUGCUpdate), "ugc update")
	}

	guild := cc.GetGuild(ownerGuildId)
	return cc.PermissionCheck(guild, activePlayer, types.PermGuildUGCUpdate)
}
