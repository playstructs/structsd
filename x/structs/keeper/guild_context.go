package keeper



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
