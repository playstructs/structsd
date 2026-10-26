package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const VaultOwnerKey = "Vault/owner/"

// VaultAddress derives a protocol account distinct from every signing address
// and collateral pool. The owner ID remains stable when a guild changes hands
// or a player rotates their primary key. Invalid aliases never derive a vault.
func VaultAddress(ownerID string) (sdk.AccAddress, error) {
	if _, _, err := ParseAssetOwnerID(ownerID); err != nil {
		return nil, err
	}
	return authtypes.NewModuleAddress(ModuleName + "/vault/" + ownerID), nil
}
