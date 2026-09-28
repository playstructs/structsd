package types

// ValidateVaultOwners checks only the vault projection so direct module imports
// can enforce it even when callers have not run the full genesis validator.
func (gs GenesisState) ValidateVaultOwners() error {
	if len(gs.VaultOwnerList) == 0 {
		return nil
	}
	owners := make(map[string]bool, len(gs.PlayerList)+len(gs.GuildList))
	for _, player := range gs.PlayerList {
		kind, _, err := ParseAssetOwnerID(player.Id)
		owners[player.Id] = err == nil && kind == ObjectType_player
	}
	for _, guild := range gs.GuildList {
		kind, _, err := ParseAssetOwnerID(guild.Id)
		owners[guild.Id] = err == nil && kind == ObjectType_guild
	}
	seen := make(map[string]bool, len(gs.VaultOwnerList))
	for _, id := range gs.VaultOwnerList {
		if _, _, err := ParseAssetOwnerID(id); err != nil {
			return err
		}
		if !owners[id] {
			return NewObjectNotFoundError("vault_owner", id)
		}
		if seen[id] {
			return NewParameterValidationError("vault_owner", 0, "duplicate")
		}
		seen[id] = true
	}
	return nil
}
