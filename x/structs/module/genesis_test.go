package structs_test

import (
	"fmt"
	"sort"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	keeperlib "structs/x/structs/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis_RebuildsProviderPoolAddressIndex(t *testing.T) {
	genesisState := types.DefaultGenesis()
	genesisState.ProviderList = []types.Provider{{Id: "10-9", Index: 9}}

	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, *genesisState)

	collateral := keeperlib.GetProviderCollateralPoolLocation("10-9").String()
	providerId, kind, found := k.GetProviderPoolAddress(ctx, collateral)
	require.True(t, found)
	require.Equal(t, "10-9", providerId)
	require.Equal(t, keeperlib.ProviderPoolKindCollateral, kind)

	exported := structs.ExportGenesis(ctx, k)
	k2, ctx2 := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx2, k2, *exported)

	providerId, kind, found = k2.GetProviderPoolAddress(ctx2, collateral)
	require.True(t, found)
	require.Equal(t, "10-9", providerId)
	require.Equal(t, keeperlib.ProviderPoolKindCollateral, kind)
}

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		PortId: types.PortID,
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, genesisState)
	got := structs.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState.PortId, got.PortId)

	// this line is used by starport scaffolding # genesis/test/assert
}

// genesisAddresses builds n valid bech32 addresses whose sorted order differs
// from the order they appear in the AddressList, so that a test which passes
// can only be passing because the commit sorted them.
func genesisAddresses(n int) []*types.AddressRecord {
	records := make([]*types.AddressRecord, 0, n)
	for i := 0; i < n; i++ {
		seed := fmt.Sprintf("%d_genesis_padding_address_12345", (n-i)*7%n)
		records = append(records, &types.AddressRecord{
			Address:     sdk.AccAddress(seed).String(),
			PlayerIndex: uint64(i + 1),
		})
	}
	return records
}

// InitGenesis loads every AddressList entry into cc.addresses and then commits
// the map. Addresses with no auth account get one, which consumes the global
// account-number sequence, so committing in map order handed two validators
// different auth state from the same genesis file and a different app hash
// before the chain had produced a block.
func TestGenesis_AddressAccountNumbersAreDeterministic(t *testing.T) {
	addressList := genesisAddresses(8)

	want := make([]string, 0, len(addressList))
	for _, record := range addressList {
		want = append(want, record.Address)
	}
	sort.Strings(want)

	// Repeat with a fresh store each time: one run can hit sorted map order by
	// luck, and this is the closest a unit test gets to several validators
	// initialising from the same file.
	for round := 0; round < 10; round++ {
		genesisState := types.GenesisState{
			Params:      types.DefaultParams(),
			PortId:      types.PortID,
			AddressList: addressList,
		}
		require.NoError(t, genesisState.Validate(), "round %d: fixture should be a valid genesis", round)

		k, ctx := keepertest.StructsKeeper(t)
		accountKeeper := keepertest.AccountKeeperFrom(t, ctx)

		structs.InitGenesis(ctx, k, genesisState)

		require.Equal(t, want, accountKeeper.AccountCreationOrder(),
			"round %d: genesis must number auth accounts in sorted address order", round)
	}
}

// A malformed address must be caught by genesis validation, so `structsd genesis
// validate` fails on the file rather than a node starting on it. This is the
// only way an unparseable address can reach SetPlayerIndexForAddress at all:
// every transaction path derives its address from a pubkey checked against
// PubKeyToBech32, or from an already-parsed AccAddress.
func TestGenesis_ValidateRejectsMalformedAddress(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		PortId: types.PortID,
		AddressList: []*types.AddressRecord{
			{Address: sdk.AccAddress("good_padding_address_1234567890").String(), PlayerIndex: 1},
			{Address: "not-a-bech32-address", PlayerIndex: 2},
		},
	}

	err := genesisState.Validate()
	require.Error(t, err, "genesis validation must reject an unparseable AddressList entry")
	require.Contains(t, err.Error(), "not-a-bech32-address")
}

// If a file somehow bypasses validation, the bad entry must not burn an account
// number on the empty address, and the valid entries around it must still
// import normally.
func TestGenesis_MalformedAddressGetsNoAuthAccount(t *testing.T) {
	good := sdk.AccAddress("good_padding_address_1234567890").String()

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		PortId: types.PortID,
		AddressList: []*types.AddressRecord{
			{Address: good, PlayerIndex: 1},
			{Address: "not-a-bech32-address", PlayerIndex: 2},
		},
	}

	k, ctx := keepertest.StructsKeeper(t)
	accountKeeper := keepertest.AccountKeeperFrom(t, ctx)

	structs.InitGenesis(ctx, k, genesisState)

	require.Equal(t, []string{good}, accountKeeper.AccountCreationOrder(),
		"only the valid address may be given an auth account")
	require.Equal(t, uint64(1), k.GetPlayerIndexFromAddress(ctx, good),
		"the valid address must still import")
}
