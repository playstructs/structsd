package keeper

// addressCache holds an address -> playerIndex mapping with change tracking.
type AddressCache struct {
	CC          *CurrentContext
    Address     string
	PlayerIndex uint64
	Loaded      bool
	Changed     bool
	Deleted     bool
}

func (cache *AddressCache) IsChanged() bool {
	return cache.Changed
}

func (cache *AddressCache) ID() string {
	return cache.Address
}

func (cache *AddressCache) Commit() {
    if cache.Loaded && cache.Changed {
        cache.Changed = false
    	cache.CC.k.logger.Info("Updating Address Index From Cache", "address", cache.Address, "playerIndex", cache.PlayerIndex)

        if cache.Deleted {
            cache.CC.k.RevokePlayerIndexForAddress(cache.CC.ctx, cache.Address, cache.PlayerIndex)
        } else {
            // Commit cannot propagate, and CommitAll runs inside block hooks
            // that can only log and continue. The entry points guard this
            // instead: GenesisState.Validate rejects a malformed AddressList,
            // and every transaction path derives its address from a verified
            // pubkey or an already-parsed AccAddress. This is the backstop.
            if err := cache.CC.k.SetPlayerIndexForAddress(cache.CC.ctx, cache.Address, cache.PlayerIndex); err != nil {
                cache.CC.k.logger.Error("Address index not written", "address", cache.Address, "playerIndex", cache.PlayerIndex, "error", err)
            }
        }
    }
}