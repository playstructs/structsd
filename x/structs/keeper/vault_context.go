package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// reactorInfusionOwner resolves protocol vaults before ordinary delegator
// auto-registration. Vault ownership never grants a signing-key association.
func (cc *CurrentContext) reactorInfusionOwner(address sdk.AccAddress) (AssetOwner, error) {
	if ownerID, found := cc.k.GetVaultOwner(cc.ctx, address); found {
		return cc.GetExistingAssetOwner(ownerID)
	}
	return cc.UpsertPlayer(address.String()), nil
}

// GetVault is a read-only lookup even for a vault not yet provisioned. It resolves
// an existing owner first and reuses that owner's entity cache.
func (cc *CurrentContext) GetVault(ownerID string) (*VaultCache, error) {
	if cache, ok := cc.vaults[ownerID]; ok {
		return cache, nil
	}
	owner, err := cc.GetExistingAssetOwner(ownerID)
	if err != nil {
		return nil, err
	}
	address, err := types.VaultAddress(ownerID)
	if err != nil {
		return nil, err
	}
	cache := &VaultCache{CC: cc, owner: owner, address: address}
	cc.vaults[ownerID] = cache
	return cache, nil
}

// EnsureVaults provisions the named owners before any bank call or staking hook
// needs their accounts and reverse index. All inputs are resolved before writing;
// account numbers are assigned in sorted owner-ID order, regardless of input order.
// This flushes only vault provisioning, not other entity caches. The surrounding
// SDK message cache remains responsible for rolling back a failed transaction.
func (cc *CurrentContext) EnsureVaults(ownerIDs ...string) ([]sdk.AccAddress, error) {
	addresses := make([]sdk.AccAddress, len(ownerIDs))
	for i, id := range ownerIDs {
		vault, err := cc.GetVault(id)
		if err != nil {
			return nil, err
		}
		addresses[i] = vault.Address()
	}
	for _, id := range ownerIDs {
		vault := cc.vaults[id]
		_, indexed := cc.k.GetVaultOwner(cc.ctx, vault.address)
		if !indexed || cc.k.accountKeeper.GetAccount(cc.ctx, vault.address) == nil {
			vault.Changed = true
		}
	}
	commitCaches(cc.vaults)
	return addresses, nil
}

// GetAssetOwnerByAddress recognizes vaults before ordinary registered keys. In
// particular a guild vault must never fall through to player auto-registration.
// This is a subject lookup and never establishes a signer.
func (cc *CurrentContext) GetAssetOwnerByAddress(address sdk.AccAddress) (AssetOwner, error) {
	if id, found := cc.k.GetVaultOwner(cc.ctx, address); found {
		return cc.GetExistingAssetOwner(id)
	}
	player, err := cc.GetPlayerByAddress(address.String())
	if err != nil {
		return nil, err
	}
	return cc.GetExistingAssetOwner(player.ID())
}

// resolveReactorDelegator treats the named address as a subject, never a signer.
// A provisioned vault uses its live player/guild owner; registered addresses
// retain their existing player-wallet authority and signing-key ceiling.
func (cc *CurrentContext) resolveReactorDelegator(address, ownerID string, actor *PlayerCache, permission types.Permission) (sdk.AccAddress, error) {
	if address == "" || ownerID != "" {
		if ownerID == "" {
			ownerID = actor.ID()
		}
		vault, err := cc.GetVault(ownerID)
		if err != nil {
			return nil, err
		}
		if err := vault.CanUseBy(actor, permission); err != nil {
			return nil, err
		}
		if address != "" {
			explicit, err := sdk.AccAddressFromBech32(address)
			if err != nil || !explicit.Equals(vault.Address()) {
				return nil, types.NewAddressValidationError(address, "vault_owner_mismatch")
			}
		}
		// Provision before staking calls its hooks, which resolve the vault's
		// owner from the reverse index. The SDK transaction cache rolls back
		// provisioning if subsequent validation or staking fails.
		vaults, err := cc.EnsureVaults(ownerID)
		if err != nil {
			return nil, err
		}
		return vaults[0], nil
	}
	delegator, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, types.NewAddressValidationError(address, "invalid_delegator")
	}
	owner, err := cc.GetAssetOwnerByAddress(delegator)
	if err != nil {
		return nil, err
	}
	if err := cc.PermissionCheck(owner, actor, permission); err != nil {
		return nil, err
	}
	return delegator, nil
}
