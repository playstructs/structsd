package structs_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs"
	"structs/x/structs/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		PortId: types.PortID,
		ReactorList: []types.Reactor{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		ReactorCount: 2,
		SubstationList: []types.Substation{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		SubstationCount: 2,
		AllocationList: []types.Allocation{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		AllocationCount: 2,
		AllocationProposalList: []types.AllocationProposal{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		AllocationProposalCount: 2,
		GuildList: []types.Guild{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		GuildCount: 2,
		PlayerList: []types.Player{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		PlayerCount: 2,
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, *k, genesisState)
	got := structs.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState.PortId, got.PortId)

	require.ElementsMatch(t, genesisState.ReactorList, got.ReactorList)
	require.Equal(t, genesisState.ReactorCount, got.ReactorCount)
	require.ElementsMatch(t, genesisState.SubstationList, got.SubstationList)
	require.Equal(t, genesisState.SubstationCount, got.SubstationCount)
	require.ElementsMatch(t, genesisState.AllocationList, got.AllocationList)
	require.Equal(t, genesisState.AllocationCount, got.AllocationCount)
	require.ElementsMatch(t, genesisState.AllocationProposalList, got.AllocationProposalList)
	require.Equal(t, genesisState.AllocationProposalCount, got.AllocationProposalCount)
	require.ElementsMatch(t, genesisState.GuildList, got.GuildList)
	require.Equal(t, genesisState.GuildCount, got.GuildCount)
	require.ElementsMatch(t, genesisState.PlayerList, got.PlayerList)
	require.Equal(t, genesisState.PlayerCount, got.PlayerCount)
	// this line is used by starport scaffolding # genesis/test/assert
}
