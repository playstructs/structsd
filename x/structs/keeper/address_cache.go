package keeper

// addressCache holds an address -> playerIndex mapping with change tracking.
//
// The registration-proof nonce is tracked alongside it but with its own loaded
// and changed flags, because the two rows have different lifetimes: revoking an
// address deletes the association and keeps the nonce.
type AddressCache struct {
	CC          *CurrentContext
    Address     string
	PlayerIndex uint64
	Loaded      bool
	Changed     bool
	Deleted     bool

	ProofNonce        uint64
	ProofNonceLoaded  bool
	ProofNonceChanged bool
}

func (cache *AddressCache) IsChanged() bool {
	return cache.Changed || cache.ProofNonceChanged
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
                cache.CC.k.logger.Error("Auth account not provisioned for address", "address", cache.Address, "playerIndex", cache.PlayerIndex, "error", err)
            }
        }
    }

    // Written whether or not the association changed, and whether or not it was
    // deleted above: the nonce outlives the association on purpose.
    if cache.ProofNonceChanged {
        cache.ProofNonceChanged = false
        cache.CC.k.SetAddressProofNonce(cache.CC.ctx, cache.Address, cache.ProofNonce)
    }
}