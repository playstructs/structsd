package keeper

// GetPlayerIndexFromAddress returns the player index for an address, caching the result.
// Returns 0 if the address is not registered or has been deleted.
func (cc *CurrentContext) GetPlayerIndexFromAddress(address string) uint64 {
	if cache, exists := cc.addresses[address]; exists && cache.Loaded {
		if cache.Deleted {
			return 0
		}
		return cache.PlayerIndex
	}

	playerIndex := cc.k.GetPlayerIndexFromAddress(cc.ctx, address)
	cc.addresses[address] = &AddressCache{
	    CC:          cc,
	    Address:     address,
		PlayerIndex: playerIndex,
		Loaded:      true,
	}
	return playerIndex
}

// SetPlayerIndexForAddress sets the player index for an address (commits during CommitAll).
// If the address was previously deleted, setting clears the deletion.
func (cc *CurrentContext) SetPlayerIndexForAddress(address string, playerIndex uint64) {

	cache, exists := cc.addresses[address]
	if !exists {
		cache = &AddressCache{}
		cc.addresses[address] = cache

        cache.CC = cc
        cache.Address = address
	}
	cache.PlayerIndex = playerIndex
	cache.Loaded = true
	cache.Changed = true
	cache.Deleted = false
}

// RevokePlayerIndexForAddress marks an address as revoked (commits during CommitAll).
func (cc *CurrentContext) RevokePlayerIndexForAddress(address string, playerIndex uint64) {
	cc.addresses[address] = &AddressCache{
	    CC:          cc,
	    Address:     address,
		PlayerIndex: playerIndex,
		Loaded:      true,
		Changed:     true,
		Deleted:     true,
	}
}
