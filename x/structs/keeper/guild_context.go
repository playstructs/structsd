package keeper

import (
	"cosmossdk.io/math"
	"structs/x/structs/types"
)

// GetGuild returns a GuildCache by ID, loading from store if not already cached.
func (cc *CurrentContext) GetGuild(guildId string) *GuildCache {
	if cache, exists := cc.guilds[guildId]; exists {
		return cache
	}

	cc.guilds[guildId] = &GuildCache{
                          		GuildId: guildId,
                          		CC: cc,
                          		Changed: false,
                          	}

	return cc.guilds[guildId]
}

func (cc *CurrentContext) GenesisImportGuild(guild types.Guild) {
	// State exported by a pre-v0.21.0 binary carries no bank fee fields, so the
	// non-nullable LegacyDec members decode as nil and would panic when the
	// cache commits (marshal). Normalize to zero on import; the upgrade
	// migration never runs on the genesis path.
	if guild.BankConvertInFee.IsNil() {
		guild.BankConvertInFee = math.LegacyZeroDec()
	}
	if guild.BankConvertOutFee.IsNil() {
		guild.BankConvertOutFee = math.LegacyZeroDec()
	}

	cache := cc.GetGuild(guild.Id)
	cache.Guild = guild
	cache.GuildLoaded = true
	cache.Changed = true
}
