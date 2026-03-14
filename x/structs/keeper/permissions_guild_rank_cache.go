package keeper

import (
	"structs/x/structs/types"
)

type GuildRankRegisterCache struct {
	CC       *CurrentContext
	ObjectId string
	GuildId  string
	Original [types.PermissionBitCount]uint64
	Register [types.PermissionBitCount]uint64
	Loaded   bool
	Changed  bool
}

func (cache *GuildRankRegisterCache) ID() string {
	return cache.ObjectId + "/" + cache.GuildId
}

func (cache *GuildRankRegisterCache) Commit() {
	if !cache.Loaded || !cache.Changed {
		return
	}
	cache.Changed = false

	var changedBits types.Permission
	for bit := 0; bit < types.PermissionBitCount; bit++ {
		if cache.Register[bit] != cache.Original[bit] {
			changedBits |= types.Permission(1 << bit)
		}
	}
	if changedBits == 0 {
		return
	}

	cache.CC.k.SetGuildRankPermission(cache.CC.ctx, cache.ObjectId, cache.GuildId, cache.Register, changedBits)
	cache.Original = cache.Register
}
