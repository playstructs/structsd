package app_test

import (
	"encoding/binary"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestReactorCommissionRequiresSuccessfulLatestSlashRepair(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	primary, _, ownerID := newTransferPlayer(t, bApp, ctx)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(1000000))
	id := jailGateReactorId(t, bApp, ctx, validator)

	assertGated := func(issued uint64) {
		t.Helper()
		revision, err := bApp.StructsKeeper.GetReactorCommissionRevision(ctx, id)
		require.NoError(t, err)
		require.Equal(t, keeperlib.ReactorCommissionRevision{Issued: issued}, revision)
		probe := bApp.StructsKeeper.NewCurrentContext(ctx)
		_, err = probe.NewAllocation(types.AllocationType_dynamic, id, "", primary.String(), ownerID, 1)
		require.ErrorContains(t, err, "unsupported_supply")
	}
	slash := func() {
		t.Helper()
		value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
		require.NoError(t, err)
		consensus, err := value.GetConsAddr()
		require.NoError(t, err)
		_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
		require.NoError(t, err)
	}
	slash()
	assertGated(1)
	store := ctx.KVStore(bApp.GetKey(types.StoreKey))
	snapshotKey := []byte("Reactor/repairSnapshot/" + id)
	firstSnapshot := append([]byte(nil), store.Get(snapshotKey)...)
	original, found := bApp.StructsKeeper.GetInfusion(ctx, id, primary.String())
	require.True(t, found)
	broken := original
	broken.DestinationType = types.ObjectType(500)
	bApp.StructsKeeper.SetInfusion(ctx, broken)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	assertGated(1)
	require.Len(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx), 1)
	failedSnapshot := append([]byte(nil), store.Get(snapshotKey)...)
	bApp.StructsKeeper.EnqueueReactorRestart(ctx, id)
	require.Equal(t, failedSnapshot, store.Get(snapshotKey), "manual restart preserves failed progress and revision")

	slash()
	assertGated(2)
	secondSnapshot := append([]byte(nil), store.Get(snapshotKey)...)
	require.NotEqual(t, firstSnapshot, secondSnapshot)
	// Simulate an obsolete saved cursor: neither its successful rows nor its
	// completion may authorize power for the newer slash.
	store.Set(snapshotKey, firstSnapshot)
	before := bApp.StructsKeeper.GetAllGridExport(ctx)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	assertGated(2)
	require.Equal(t, before, bApp.StructsKeeper.GetAllGridExport(ctx))
	require.Len(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx), 1)
	store.Set(snapshotKey, secondSnapshot)
	bApp.StructsKeeper.SetInfusion(ctx, original)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	revision, err := bApp.StructsKeeper.GetReactorCommissionRevision(ctx, id)
	require.NoError(t, err)
	require.Equal(t, keeperlib.ReactorCommissionRevision{Issued: 2, Certified: 2}, revision)
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	require.False(t, store.Has(snapshotKey))
	value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	// Both positions have equal shares and carry 4%
	// commission. These divisible fixture values need no per-position rounding.
	capacity := cc.GetGridAttribute(keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, id))
	require.Equal(t, value.Tokens.Uint64()*4/100, capacity)
	_, err = cc.NewAllocation(types.AllocationType_dynamic, id, "", primary.String(), ownerID, 1)
	require.NoError(t, err)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestReactorCommissionRevisionRejectsCorruptionAndExhaustion(t *testing.T) {
	encode := func(issued, certified uint64) []byte {
		value := make([]byte, 16)
		binary.BigEndian.PutUint64(value[:8], issued)
		binary.BigEndian.PutUint64(value[8:], certified)
		return value
	}
	for _, tc := range []struct {
		name  string
		value []byte
	}{
		{"malformed", []byte{1}},
		{"future-certificate", encode(1, 2)},
		{"exhausted", encode(^uint64(0), ^uint64(0))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			k := bApp.StructsKeeper
			const id = "3-1"
			validator := sdk.ValAddress("revisionvalidator123")
			k.SetReactor(ctx, types.Reactor{Id: id, Validator: validator.String()})
			k.SetReactorValidatorBytes(ctx, id, validator)

			store := ctx.KVStore(bApp.GetKey(types.StoreKey))
			key := []byte("Reactor/commissionRevision/" + id)
			store.Set(key, tc.value)
			if tc.name == "exhausted" {
				require.Panics(t, func() { _ = k.ReactorUpdateInfusionsFromSlashing(ctx, validator, math.LegacyNewDecWithPrec(5, 2)) })
			} else {
				require.Error(t, k.ReactorUpdateInfusionsFromSlashing(ctx, validator, math.LegacyNewDecWithPrec(5, 2)))
			}
			require.Equal(t, tc.value, store.Get(key), "invalid state must not wrap or lose a pending revision")
			require.Empty(t, k.GetReactorSlashReconcileQueue(ctx))
		})
	}
}
