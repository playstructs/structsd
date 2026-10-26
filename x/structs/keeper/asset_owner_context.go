package keeper

import "structs/x/structs/types"

// AssetOwner is the permission-bearing player or guild whose assets an action
// uses. The actor remains a PlayerCache authenticated on CurrentContext; an
// asset owner is never an alternative signing identity.
type AssetOwner interface {
	PermissionedObject
}

// GetExistingAssetOwner validates type and existence before returning the same
// entity cache used by the player/guild accessors. It allocates no parallel cache
// and cannot create a player for a guild identity.
func (cc *CurrentContext) GetExistingAssetOwner(id string) (AssetOwner, error) {
	kind, _, err := types.ParseAssetOwnerID(id)
	if err != nil {
		return nil, err
	}
	switch kind {
	case types.ObjectType_player:
		player, err := cc.GetExistingPlayer(id)
		if err != nil {
			return nil, err
		}
		return player, nil
	case types.ObjectType_guild:
		guild := cc.GetGuild(id)
		if err := guild.CheckGuild(); err != nil {
			return nil, err
		}
		return guild, nil
	default:
		return nil, types.NewObjectNotFoundError("asset_owner", id)
	}
}
