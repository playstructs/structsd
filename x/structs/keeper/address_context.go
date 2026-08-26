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
	cache := cc.addressCache(address)
	cache.PlayerIndex = playerIndex
	cache.Loaded = true
	return playerIndex
}

// addressCache returns the cache entry for an address, creating an empty one if
// this is the first time the address is touched.
//
// Every entry point goes through this rather than assigning a fresh struct,
// because the association and the proof nonce are tracked on the same entry with
// separate flags. Replacing the struct would silently drop whichever of the two
// the caller was not writing.
func (cc *CurrentContext) addressCache(address string) *AddressCache {
	if cache, exists := cc.addresses[address]; exists {
		return cache
	}

	cache := &AddressCache{CC: cc, Address: address}
	cc.addresses[address] = cache
	return cache
}

// GetAddressProofNonce returns the registration-proof nonce for an address,
// caching the result. An address that has never registered starts at 0.
func (cc *CurrentContext) GetAddressProofNonce(address string) uint64 {
	cache := cc.addressCache(address)
	if !cache.ProofNonceLoaded {
		cache.ProofNonce = cc.k.GetAddressProofNonce(cc.ctx, address)
		cache.ProofNonceLoaded = true
	}
	return cache.ProofNonce
}

// IncrementAddressProofNonce burns the nonce a registration proof was signed
// against, so that proof can never be presented again.
func (cc *CurrentContext) IncrementAddressProofNonce(address string) uint64 {
	next := cc.GetAddressProofNonce(address) + 1

	cache := cc.addressCache(address)
	cache.ProofNonce = next
	cache.ProofNonceChanged = true

	return next
}

// GenesisImportAddressNonce stages a genesis proof nonce for CommitAll.
func (cc *CurrentContext) GenesisImportAddressNonce(address string, nonce uint64) {
	cache := cc.addressCache(address)
	cache.ProofNonce = nonce
	cache.ProofNonceLoaded = true
	cache.ProofNonceChanged = true
}

// GenesisImportAddress stages a genesis address association for CommitAll.
//
// A malformed address is rejected upstream by GenesisState.Validate, and
// SetPlayerIndexForAddress refuses to provision an auth account for one, so
// there is nothing left for this to check.
func (cc *CurrentContext) GenesisImportAddress(address string, playerIndex uint64) {
	cache := cc.addressCache(address)
	cache.PlayerIndex = playerIndex
	cache.Loaded = true
	cache.Changed = true
	cache.Deleted = false
}

// SetPlayerIndexForAddress sets the player index for an address (commits during CommitAll).
// If the address was previously deleted, setting clears the deletion.
func (cc *CurrentContext) SetPlayerIndexForAddress(address string, playerIndex uint64) {

	cache := cc.addressCache(address)
	cache.PlayerIndex = playerIndex
	cache.Loaded = true
	cache.Changed = true
	cache.Deleted = false
}

// RevokePlayerIndexForAddress marks an address as revoked (commits during CommitAll).
func (cc *CurrentContext) RevokePlayerIndexForAddress(address string, playerIndex uint64) {
	cache := cc.addressCache(address)
	cache.PlayerIndex = playerIndex
	cache.Loaded = true
	cache.Changed = true
	cache.Deleted = true
}
