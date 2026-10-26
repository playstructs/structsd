package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestReactorCommissionRevisionOnlyAdvancesForPositiveSlash(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const id = "3-1"
	validator := sdk.ValAddress("revisionvalidator123")
	k.SetReactor(ctx, types.Reactor{Id: id, Validator: validator.String()})
	k.SetReactorValidatorBytes(ctx, id, validator)
	positive := math.LegacyNewDecWithPrec(5, 2)

	for _, fraction := range []math.LegacyDec{{}, math.LegacyZeroDec(), math.LegacyNewDec(-1)} {
		require.NoError(t, k.ReactorUpdateInfusionsFromSlashing(ctx, validator, fraction))
		revision, err := k.GetReactorCommissionRevision(ctx, id)
		require.NoError(t, err)
		require.Equal(t, keeperlib.ReactorCommissionRevision{}, revision)
	}
	require.NoError(t, k.ReactorUpdateInfusionsFromSlashing(ctx, validator, positive))
	snapshot, found, err := k.GetReactorRepairSnapshot(ctx, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(1), snapshot.Revision)
	k.EnqueueReactorRestart(ctx, id)
	k.EnqueueReactorSlashReconcile(ctx, id) // a new repair pass is not a new slash
	revision, err := k.GetReactorCommissionRevision(ctx, id)
	require.NoError(t, err)
	require.Equal(t, keeperlib.ReactorCommissionRevision{Issued: 1}, revision)
	k.ProcessReactorSlashReconcileQueue(ctx)
	revision, err = k.GetReactorCommissionRevision(ctx, id)
	require.NoError(t, err)
	require.Equal(t, keeperlib.ReactorCommissionRevision{Issued: 1, Certified: 1}, revision)
	require.NoError(t, k.ReactorUpdateInfusionsFromSlashing(ctx, validator, positive))
	revision, err = k.GetReactorCommissionRevision(ctx, id)
	require.NoError(t, err)
	require.Equal(t, keeperlib.ReactorCommissionRevision{Issued: 2, Certified: 1}, revision)
}
