package keeper

import (
	"structs/x/structs/types"
)

type PermissionsGuildRankCache struct {
	CC                   *CurrentContext
	PermissionGuildRankID string
	ObjectId              string
	GuildId               string
	Permission            types.Permission
	HighestRank           uint64
	Loaded                bool
	Changed               bool
	Deleted               bool
	Exists                bool // true if a record exists (from store or just set); false when not found or revoked
}


func (cache *PermissionsGuildRankCache) IsChanged() bool {
	return cache.Changed
}

func (cache *PermissionsGuildRankCache) ID() string {
	return cache.PermissionGuildRankID
}

func (cache *PermissionsGuildRankCache) Commit() {
    if cache.Loaded && cache.Changed {
        cache.Changed = false
    	cache.CC.k.logger.Info("Updating Guild Rank Permission From Cache", "PermissionsId", cache.ID(), "value", cache.HighestRank)

        if cache.Deleted {
            cache.CC.k.RemoveGuildRankPermission(cache.CC.ctx, cache.ObjectId, cache.GuildId, cache.Permission)
        } else {
            cache.CC.k.SetHighestGuildRankPermission(cache.CC.ctx, cache.ObjectId, cache.GuildId, cache.Permission, cache.HighestRank)
        }
    }
}