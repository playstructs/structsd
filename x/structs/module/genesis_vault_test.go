package structs_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"
)

func vaultGenesisFixture() *types.GenesisState {
	state := types.DefaultGenesis()
	state.PlayerList = []types.Player{{Id: "1-1", Index: 1}, {Id: "1-2", Index: 2}}
	state.PlayerCount = 3
	guild := types.CreateEmptyGuild()
	guild.Id = "0-1"
	guild.Index = 1
	guild.Owner = "1-1"
	state.GuildList = []types.Guild{guild}
	state.GuildCount = 2
	state.VaultOwnerList = []string{"1-1", "0-1"}
	return state
}

func TestGenesisVaultsRoundTripAndPreserveAuthAccountNumbers(t *testing.T) {
	state := vaultGenesisFixture()
	require.NoError(t, state.ValidateVaultOwners())
	k, ctx := keepertest.StructsKeeper(t)
	address, err := types.VaultAddress("1-1")
	require.NoError(t, err)
	// Auth genesis precedes Structs genesis in the app. A restored account must
	// retain its original number rather than consume the sequence again.
	k.AccountKeeper().SetAccount(ctx, authtypes.NewBaseAccount(address, nil, 77, 0))
	structs.InitGenesis(ctx, k, *state)
	require.Equal(t, uint64(77), k.AccountKeeper().GetAccount(ctx, address).GetAccountNumber())
	for _, id := range state.VaultOwnerList {
		address, err := types.VaultAddress(id)
		require.NoError(t, err)
		owner, found := k.GetVaultOwner(ctx, address)
		require.True(t, found)
		require.Equal(t, id, owner)
		require.Zero(t, k.GetPlayerIndexFromAddress(ctx, address.String()))
	}
	inactive, err := types.VaultAddress("1-2")
	require.NoError(t, err)
	require.Nil(t, k.AccountKeeper().GetAccount(ctx, inactive), "unprovisioned owners stay unprovisioned")
	exported := structs.ExportGenesis(ctx, k)
	require.Equal(t, []string{"0-1", "1-1"}, exported.VaultOwnerList)
	k2, ctx2 := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx2, k2, *exported)
	again := structs.ExportGenesis(ctx2, k2)
	require.Equal(t, exported.VaultOwnerList, again.VaultOwnerList)
	for _, id := range again.VaultOwnerList {
		address, err := types.VaultAddress(id)
		require.NoError(t, err)
		_, found := k2.GetVaultOwner(ctx2, address)
		require.True(t, found)
	}
}

func TestGenesisRejectsInvalidVaultOwnersBeforeImport(t *testing.T) {
	for _, ids := range [][]string{{"1-1", "1-1"}, {"1-999"}, {"0-999"}, {"3-1"}, {"1-01"}} {
		state := vaultGenesisFixture()
		state.VaultOwnerList = ids
		require.Error(t, state.ValidateVaultOwners())
		k, ctx := keepertest.StructsKeeper(t)
		require.Panics(t, func() { structs.InitGenesis(ctx, k, *state) })
		_, found := k.GetPlayer(ctx, "1-1")
		require.False(t, found)
	}
}

func TestGenesisVaultProvisioningOrderDoesNotDependOnInputOrder(t *testing.T) {
	for _, ids := range [][]string{{"1-1", "0-1"}, {"0-1", "1-1"}} {
		state := vaultGenesisFixture()
		state.VaultOwnerList = ids
		k, ctx := keepertest.StructsKeeper(t)
		structs.InitGenesis(ctx, k, *state)
		guild, _ := types.VaultAddress("0-1")
		player, _ := types.VaultAddress("1-1")
		require.Less(t, k.AccountKeeper().GetAccount(ctx, guild).GetAccountNumber(), k.AccountKeeper().GetAccount(ctx, player).GetAccountNumber())
	}
}
