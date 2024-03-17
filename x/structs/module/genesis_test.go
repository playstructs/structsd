package structs_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	structs "structs/x/structs/module"
	"structs/x/structs/types"

	"github.com/stretchr/testify/require"
)

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
