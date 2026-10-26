package v0_22_0_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestVaultAddressPlanUsesCurrentAssociationsAndDoesNotProvision(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	k.SetPlayer(ctx, types.Player{Id: "1-1", Index: 1})
	k.SetPlayer(ctx, types.Player{Id: "1-2", Index: 2})
	first := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	second := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	third := sdk.AccAddress(bytes.Repeat([]byte{3}, 20))
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, second.String(), 1))
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, first.String(), 1))
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, third.String(), 2))
	// Reassignment must use the current index, not any old infusion ownership.
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, second.String(), 2))
	plan, err := v22.BuildVaultAddressMigrationPlan(ctx, &upgrades.Keepers{StructsKeeper: k})
	require.NoError(t, err)
	require.Len(t, plan, 3)
	expected := map[string]string{first.String(): "1-1", second.String(): "1-2", third.String(): "1-2"}
	for i, move := range plan {
		require.Equal(t, expected[move.Source], move.OwnerID)
		vault, err := types.VaultAddress(move.OwnerID)
		require.NoError(t, err)
		require.Equal(t, vault.String(), move.Destination)
		require.Nil(t, k.AccountKeeper().GetAccount(ctx, vault))
		_, indexed := k.GetVaultOwner(ctx, vault)
		require.False(t, indexed)
		if i > 0 {
			require.Less(t, plan[i-1].Source, move.Source)
		}
	}
}

func TestVaultAddressPlanRejectsOrphanedAndProtocolSources(t *testing.T) {
	for _, scenario := range []string{"orphan", "vault"} {
		t.Run(scenario, func(t *testing.T) {
			k, ctx := keepertest.StructsKeeper(t)
			address := sdk.AccAddress(bytes.Repeat([]byte{4}, 20))
			if scenario == "vault" {
				k.SetPlayer(ctx, types.Player{Id: "1-1", Index: 1})
				var err error
				address, err = types.VaultAddress("1-1")
				require.NoError(t, err)
			}
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), 1))
			plan, err := v22.BuildVaultAddressMigrationPlan(ctx, &upgrades.Keepers{StructsKeeper: k})
			require.Error(t, err)
			require.Nil(t, plan)
		})
	}
}
