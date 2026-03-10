package keeper

import (
	"structs/x/structs/types"
    "fmt"
)

type PermissionsGuildRankCache struct {
    CC              *CurrentContext
    PermissionGuildRankID string
    ObjectId        string
    PlayerId        string
    Permission      types.Permission
	HighestRank     uint64
	Loaded          bool
	Changed         bool
	Deleted         bool
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
            if cache.HighestRank > 0 {
                cache.CC.k.SetHighestGuildRankPermission(cache.CC.ctx, cache.ObjectId, cache.GuildId, cache.Permission, cache.HighestRank)
            }
        }
    }
}







