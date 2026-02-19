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
            cache.CC.k.SetPlayerIndexForAddress(cache.CC.ctx, cache.Address, cache.PlayerIndex)
        }
    }
}