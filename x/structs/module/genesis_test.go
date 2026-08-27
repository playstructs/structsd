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

// A planet exported while a fleet was visiting carries a nonzero
// LocationListCount. On import every fleet is sent home, so the count must
// reset to 0 or the planet is permanently un-raidable (MsgFleetMove rejects
// queue_full forever). LocationListExtra is real capacity config and persists.
func TestGenesis_ResetsPlanetLocationListCount(t *testing.T) {
	genesisState := types.DefaultGenesis()
	genesisState.PlanetList = []types.Planet{{
		Id:                "6-1",
		LocationListStart: "9-1",
		LocationListLast:  "9-2",
		LocationListCount: 3,
		LocationListExtra: 2,
	}}

	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, *genesisState)

	planet, found := k.GetPlanet(ctx, "6-1")
	require.True(t, found)
	require.Equal(t, uint64(0), planet.LocationListCount, "visitor count must reset on import")
	require.Equal(t, "", planet.LocationListStart)
	require.Equal(t, "", planet.LocationListLast)
	require.Equal(t, uint64(2), planet.LocationListExtra, "capacity config must persist")
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

func TestGenesis_RebuildsGuildNameIndexFirstWins(t *testing.T) {
	first := types.CreateEmptyGuild()
	first.Id = "4-1"
	first.Index = 1
	first.Name = "Alpha Guild"

	second := types.CreateEmptyGuild()
	second.Id = "4-2"
	second.Index = 2
	second.Name = "alpha guild"

	genesisState := types.DefaultGenesis()
	genesisState.GuildList = []types.Guild{first, second}

	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, *genesisState)

	guildId, found := k.GetGuildIdByName(ctx, "ALPHA GUILD")
	require.True(t, found)
	require.Equal(t, first.Id, guildId)

	storedSecond, found := k.GetGuild(ctx, second.Id)
	require.True(t, found)
	require.Empty(t, storedSecond.Name,
		"the losing duplicate must not retain a name that can delete the winner's index")
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

// TestGenesis_AddressProofNoncesSurviveExportImport covers the half of the
// replay fix that only a chain restore can break.
//
// The nonce is what makes an address registration proof single-use. It is
// deliberately kept in its own store rather than on AddressRecord, because a
// revoked address keeps its nonce and loses its association — so exporting only
// the associations would silently reset exactly the addresses whose proofs are
// still floating around, and every one of them would go live again on the
// restored chain.
func TestGenesis_AddressProofNoncesSurviveExportImport(t *testing.T) {
	registered := sdk.AccAddress("registered_address_padding_1234").String()
	revoked := sdk.AccAddress("revoked_address_padding_123456").String()

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		PortId: types.PortID,
		AddressList: []*types.AddressRecord{
			{Address: registered, PlayerIndex: 1},
		},
		AddressNonceList: []*types.AddressNonceRecord{
			{Address: registered, Nonce: 3},
			// No AddressRecord: this address was revoked, which is the case the
			// separate list exists for.
			{Address: revoked, Nonce: 7},
		},
	}

	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, genesisState)

	require.Equal(t, uint64(3), k.GetAddressProofNonce(ctx, registered))
	require.Equal(t, uint64(7), k.GetAddressProofNonce(ctx, revoked),
		"a revoked address keeps its nonce, so the import has to carry it")
	require.Zero(t, k.GetPlayerIndexFromAddress(ctx, revoked),
		"and must not resurrect the association that was revoked")

	exported := structs.ExportGenesis(ctx, k)
	require.Len(t, exported.AddressNonceList, 2)

	roundTripped := make(map[string]uint64, len(exported.AddressNonceList))
	for _, record := range exported.AddressNonceList {
		roundTripped[record.Address] = record.Nonce
	}
	require.Equal(t, uint64(3), roundTripped[registered])
	require.Equal(t, uint64(7), roundTripped[revoked])
}

func TestGenesis_ValidateRejectsDuplicateAddressNonce(t *testing.T) {
	address := sdk.AccAddress("duplicate_address_padding_1234").String()

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		PortId: types.PortID,
		AddressNonceList: []*types.AddressNonceRecord{
			{Address: address, Nonce: 1},
			{Address: address, Nonce: 0},
		},
	}

	err := genesisState.Validate()
	require.Error(t, err, "two nonces for one address leaves which one wins to map order")
	require.Contains(t, err.Error(), address)
}

/* TestGenesis_RestoresProviderCheckpointBlock is the regression on a restart
 * sweeping every consumer's collateral into the provider's earnings.
 *
 * checkpointBlock is a clock, and it is the one grid attribute the import cannot
 * derive from the objects it is rebuilding. Load and capacity are excluded from
 * the grid import precisely because GenesisImportAgreement reconstructs them, so
 * admitting them would double-count - but excluding the checkpoint left it
 * unset, and an unset grid attribute reads as zero.
 *
 * Checkpoint() then bills (currentBlock - 0) * rate * aggregate load. The
 * collateral pool is shared across a provider's agreements and SweepRevenue
 * clamps to what it holds rather than failing, so the first checkpoint after a
 * restore empties it - and the money it takes is the escrow backing service
 * those consumers have not received yet.
 */
func TestGenesis_RestoresProviderCheckpointBlock(t *testing.T) {
	const providerId = "10-9"
	const exportedCheckpoint = uint64(4_000_000)

	genesisState := types.DefaultGenesis()
	genesisState.ProviderList = []types.Provider{{Id: providerId, Index: 9}}
	genesisState.GridList = []*types.GridRecord{{
		AttributeId: keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, providerId),
		Value:       exportedCheckpoint,
	}}

	k, ctx := keepertest.StructsKeeper(t)
	ctx = ctx.WithBlockHeight(int64(exportedCheckpoint) + 1)
	structs.InitGenesis(ctx, k, *genesisState)

	restored := k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, providerId))
	require.Equal(t, exportedCheckpoint, restored,
		"the exported checkpoint must survive the import; zero here bills the whole chain height")
}

/* TestGenesis_StampsMissingProviderCheckpointBlock covers the file that carries
 * no checkpoint row at all - a hand-written genesis, or one exported before the
 * attribute was imported. Zero would mean "never paid", so the provider is
 * stamped as paid up to the genesis height, which is the only reading
 * consistent with the agreements it starts with.
 */
func TestGenesis_StampsMissingProviderCheckpointBlock(t *testing.T) {
	const providerId = "10-9"

	genesisState := types.DefaultGenesis()
	genesisState.ProviderList = []types.Provider{{Id: providerId, Index: 9}}

	k, ctx := keepertest.StructsKeeper(t)
	ctx = ctx.WithBlockHeight(1_234)
	structs.InitGenesis(ctx, k, *genesisState)

	stamped := k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, providerId))
	require.Equal(t, uint64(1_234), stamped,
		"a provider with no exported checkpoint must start paid up, not from block zero")
}

// TestGenesis_ProviderCheckpointSurvivesRoundTrip is the property the two tests
// above are really about: export then import must not move the clock.
func TestGenesis_ProviderCheckpointSurvivesRoundTrip(t *testing.T) {
	const providerId = "10-9"

	genesisState := types.DefaultGenesis()
	genesisState.ProviderList = []types.Provider{{Id: providerId, Index: 9}}

	k, ctx := keepertest.StructsKeeper(t)
	ctx = ctx.WithBlockHeight(900_000)
	structs.InitGenesis(ctx, k, *genesisState)

	attributeId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, providerId)
	first := k.GetGridAttribute(ctx, attributeId)
	require.Equal(t, uint64(900_000), first)

	exported := structs.ExportGenesis(ctx, k)

	k2, ctx2 := keepertest.StructsKeeper(t)
	ctx2 = ctx2.WithBlockHeight(900_001)
	structs.InitGenesis(ctx2, k2, *exported)

	require.Equal(t, first, k2.GetGridAttribute(ctx2, attributeId),
		"a restart one block later must resume the clock, not restart it")
}
