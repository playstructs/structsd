package keeper_test

import (
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
	"testing"
)

func TestRuntimeParamsCannotRepriceDiversity(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	original := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, original))
	for _, maximum := range []bool{true, false} {
		updated := original
		if maximum {
			updated.DiversityMaxBonusBps++
		} else {
			updated.DiversityTargetExtraReactors++
		}
		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: updated})
		require.ErrorContains(t, err, "require an upgrade")
		require.Equal(t, original, k.GetParams(ctx))
	}
	updated := original
	updated.ReactorStakeCapBps = 2500
	updated.ReactorStakeCapMinValidators = 5
	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: updated})
	require.NoError(t, err)
	require.Equal(t, updated, k.GetParams(ctx))
}
