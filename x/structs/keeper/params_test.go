package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.StructsKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
