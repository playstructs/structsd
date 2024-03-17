package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
