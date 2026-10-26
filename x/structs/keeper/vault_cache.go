package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// VaultCache holds the protocol address of an existing owner. Balances remain
// authoritative in the bank keeper; caching them here would miss SDK transfers.
// No address permission row or signing-player association is created for a vault.
type VaultCache struct {
	CC      *CurrentContext
	owner   AssetOwner
	address sdk.AccAddress
	Changed bool
}

func (cache *VaultCache) ID() string      { return cache.owner.ID() }
func (cache *VaultCache) IsChanged() bool { return cache.Changed }
func (cache *VaultCache) Address() sdk.AccAddress {
	return append(sdk.AccAddress(nil), cache.address...)
}
func (cache *VaultCache) Owner() AssetOwner { return cache.owner }

func (cache *VaultCache) Commit() {
	if !cache.Changed {
		return
	}
	k := cache.CC.k
	account := k.accountKeeper.GetAccount(cache.CC.ctx, cache.address)
	created := account == nil
	if created {
		account = k.accountKeeper.NewAccountWithAddress(cache.CC.ctx, cache.address)
		k.accountKeeper.SetAccount(cache.CC.ctx, account)
	}
	k.setVaultOwner(cache.CC.ctx, cache.address, cache.ID())
	_ = sdk.UnwrapSDKContext(cache.CC.ctx).EventManager().EmitTypedEvent(&types.EventVaultAddress{
		EventVaultAddressDetail: &types.EventVaultAddressDetail{
			OwnerId: cache.ID(), VaultAddress: cache.address.String(),
			AccountNumber: account.GetAccountNumber(), AccountCreated: created,
		},
	})
	cache.Changed = false
}

// CanUseBy applies both the signing-key ceiling and authority on the represented
// owner. Looking up a guild vault is never itself authority to spend its funds.
func (cache *VaultCache) CanUseBy(actor *PlayerCache, permission types.Permission) error {
	return cache.CC.PermissionCheck(cache.owner, actor, permission)
}
