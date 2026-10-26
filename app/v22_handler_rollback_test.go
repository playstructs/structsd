package app_test

import (
	"encoding/binary"
	"errors"
	"testing"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestV22HandlerLateFailureRollsBackAssetsRepairsAndEvents(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, _, playerID := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(1000000))
	fundRedelegationAccount(t, bApp, ctx, primary, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 10)))
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	infusion := cc.GetInfusion(reactorID, primary.String())
	infusion.SetFuel(infusion.GetFuel() + 100)
	cc.CommitAll()
	// Fuel repair and asset relocation succeed before the final event snapshot
	// encounters an invalid commission revision. No earlier writes may escape.
	ctx.KVStore(bApp.GetKey(types.StoreKey)).Set([]byte("Reactor/commissionRevision/"+reactorID), []byte{1})
	bApp.StructsKeeper.GetGridCascadeQueue(ctx, true)
	beforeGrid := bApp.StructsKeeper.GetAllGridExport(ctx)
	beforeInfusions := bApp.StructsKeeper.GetAllInfusion(ctx)
	beforeBalance := bApp.BankKeeper.GetAllBalances(ctx, primary)
	beforeDelegation, err := bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
	require.NoError(t, err)
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	probe, _ := ctx.CacheContext()
	require.NoError(t, v22.MigrateVaultAssets(probe, delegationUpgradeKeepers(bApp)), "asset phase must succeed before the intended late failure")
	require.NotEmpty(t, bApp.BankKeeper.GetAllBalances(probe, vault))
	handler := v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))
	_, err = handler(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, bApp.ModuleManager.GetVersionMap())
	require.ErrorContains(t, err, "invalid reactor commission revision")
	require.Empty(t, ctx.EventManager().Events())

	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, vault))
	_, found := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
	require.False(t, found)
	require.Equal(t, beforeBalance, bApp.BankKeeper.GetAllBalances(ctx, primary))
	afterDelegation, err := bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
	require.NoError(t, err)
	require.Equal(t, beforeDelegation, afterDelegation)
	require.Equal(t, beforeInfusions, bApp.StructsKeeper.GetAllInfusion(ctx))
	require.Equal(t, beforeGrid, bApp.StructsKeeper.GetAllGridExport(ctx))
	require.Empty(t, bApp.StructsKeeper.GetGridCascadeQueue(ctx, false))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestV22HandlerModuleMigrationFailureRollsBackCompletedEconomy(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	const orphanPower = "5-6-999999"
	bApp.StructsKeeper.SetGridAttribute(ctx, orphanPower, 123)
	primary, _, playerID := newTransferPlayer(t, bApp, ctx)
	coins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 10))
	fundRedelegationAccount(t, bApp, ctx, primary, coins)
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	versions := bApp.ModuleManager.GetVersionMap()
	require.Greater(t, versions[banktypes.ModuleName], uint64(1))
	versions[banktypes.ModuleName]--
	// These indexes are derived state and therefore absent from the public
	// genesis export. Inspect their raw keys so export equality cannot hide a
	// partial rebuild escaping the outer upgrade cache.
	planet := types.Planet{Id: "2-999999", Owner: playerID, Status: types.PlanetStatus_active}
	planetValue := bApp.AppCodec().MustMarshal(&planet)
	permissionID := "6-999999@" + playerID
	permissionValue := make([]byte, 8)
	binary.BigEndian.PutUint64(permissionValue, uint64(types.PermDelete))
	store := ctx.KVStore(bApp.GetKey(types.StoreKey))
	store.Set([]byte(types.PlanetKey+planet.Id), planetValue)
	store.Set([]byte(types.PermissionKey+permissionID), permissionValue)
	infusions := bApp.StructsKeeper.GetAllInfusion(ctx)
	require.NotEmpty(t, infusions)
	retained := infusions[0]
	require.NotEmpty(t, retained.AssetOwnerID())
	infusionIndexKey := []byte("Infusion/owner/" + retained.AssetOwnerID() + "/" + retained.DestinationId + "/" + retained.Address)
	store.Delete(infusionIndexKey) // simulate legacy state without this derived index
	orderAddressKey := []byte("Reactor/infusionOrder/address/" + retained.DestinationId + "/" + retained.Address)
	oldOrdinal := append([]byte(nil), store.Get(orderAddressKey)...)
	require.Len(t, oldOrdinal, 8)
	oldEntryKey := append([]byte("Reactor/infusionOrder/entry/"+retained.DestinationId+"/"), oldOrdinal...)
	store.Delete(orderAddressKey)
	store.Delete(oldEntryKey)
	repairKey := []byte("Reactor/repairSnapshot/" + retained.DestinationId)
	legacyRepair := make([]byte, 16)
	binary.BigEndian.PutUint64(legacyRepair[:8], 99)
	binary.BigEndian.PutUint64(legacyRepair[8:], 100)
	queueKey := []byte(types.ReactorSlashReconcileQueue + retained.DestinationId)
	store.Set(repairKey, legacyRepair)
	store.Set(queueKey, []byte("legacy-cursor"))

	// Retained burned fuel must gain an insertion index and finite cleanup job
	// inside the same outer transaction, including when its generator is gone.
	const generatorID = "5-999997"
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	generatorInfusion := cc.UpsertInfusion(types.ObjectType_struct, generatorID, primary.String(), playerID)
	generatorInfusion.SetRatio(2)
	generatorInfusion.AddFuel(7)
	cc.CommitAll()
	generatorOrderKey := []byte("Reactor/infusionOrder/address/" + generatorID + "/" + primary.String())
	generatorOrdinal := append([]byte(nil), store.Get(generatorOrderKey)...)
	require.Len(t, generatorOrdinal, 8)
	generatorEntryKey := append([]byte("Reactor/infusionOrder/entry/"+generatorID+"/"), generatorOrdinal...)
	store.Delete(generatorOrderKey)
	store.Delete(generatorEntryKey)
	generatorQueueKey := []byte("Generator/powerQueue/" + generatorID)
	store.Set(generatorQueueKey, legacyRepair)

	newKeys := [][]byte{
		infusionIndexKey,
		[]byte("Planet/owner/" + playerID + "/" + planet.Id),
		[]byte("Permission/player/" + playerID + "\x00" + permissionID),
	}
	staleKeys := [][]byte{
		[]byte("Reactor/infusionOrder/stale/unused"),
		[]byte("Infusion/owner/1-999999/3-999999/stale"),
		[]byte("Planet/owner/" + playerID + "/2-999998"),
		[]byte("Permission/player/" + playerID + "\x006-999998@" + playerID),
	}
	for _, key := range newKeys {
		require.Nil(t, store.Get(key))
	}
	for _, key := range staleKeys {
		store.Set(key, []byte{7})
	}
	assertIndexState := func(testCtx sdk.Context, rebuilt bool) {
		t.Helper()
		testStore := testCtx.KVStore(bApp.GetKey(types.StoreKey))
		powerRow := testStore.Get([]byte(types.GridAttributeKey + orphanPower))
		if rebuilt {
			require.Nil(t, powerRow, "orphan repair must precede module migration")
		} else {
			require.Len(t, powerRow, 8, "a later failure must restore the deleted row")
			require.Equal(t, uint64(123), binary.BigEndian.Uint64(powerRow))
		}
		for _, key := range newKeys {
			if rebuilt {
				require.Equal(t, []byte{1}, testStore.Get(key))
			} else {
				require.Nil(t, testStore.Get(key))
			}
		}
		for _, key := range staleKeys {
			if rebuilt {
				require.Nil(t, testStore.Get(key))
			} else {
				require.Equal(t, []byte{7}, testStore.Get(key))
			}
		}
		if rebuilt {
			ordinal := testStore.Get(orderAddressKey)
			require.Len(t, ordinal, 8)
			require.Equal(t, []byte(retained.Address), testStore.Get(append([]byte("Reactor/infusionOrder/entry/"+retained.DestinationId+"/"), ordinal...)))
			repair, found, err := bApp.StructsKeeper.GetReactorRepairSnapshot(testCtx, retained.DestinationId)
			require.NoError(t, err)
			require.True(t, found)
			require.Zero(t, repair.After)
			require.Equal(t, bApp.StructsKeeper.ReactorInfusionOrderEnd(testCtx, retained.DestinationId), repair.Through)
			require.Zero(t, repair.Revision)
			require.Empty(t, testStore.Get(queueKey))
			generatorOrdinal := testStore.Get(generatorOrderKey)
			require.Len(t, generatorOrdinal, 8)
			require.Equal(t, []byte(primary.String()), testStore.Get(append([]byte("Reactor/infusionOrder/entry/"+generatorID+"/"), generatorOrdinal...)))
			generatorSnapshot, queued, err := bApp.StructsKeeper.GetGeneratorPowerSnapshot(testCtx, generatorID)
			require.NoError(t, err)
			require.True(t, queued)
			require.Zero(t, generatorSnapshot.After)
			require.Equal(t, uint64(1), generatorSnapshot.Through)
		} else {
			require.Nil(t, testStore.Get(orderAddressKey))
			require.Nil(t, testStore.Get(oldEntryKey))
			require.Equal(t, legacyRepair, testStore.Get(repairKey))
			require.Equal(t, []byte("legacy-cursor"), testStore.Get(queueKey))
			require.Nil(t, testStore.Get(generatorOrderKey))
			require.Nil(t, testStore.Get(generatorEntryKey))
			require.Equal(t, legacyRepair, testStore.Get(generatorQueueKey))
		}
		require.Equal(t, planetValue, testStore.Get([]byte(types.PlanetKey+planet.Id)))
		require.Equal(t, permissionValue, testStore.Get([]byte(types.PermissionKey+permissionID)))
	}
	config := module.NewConfigurator(bApp.AppCodec(), nil, nil)
	failure := errors.New("test module migration failure")
	called := false
	failModuleMigration := true
	require.NoError(t, config.RegisterMigration(banktypes.ModuleName, versions[banktypes.ModuleName], func(migrationCtx sdk.Context) error {
		called = true
		assertIndexState(migrationCtx, true)
		require.Equal(t, coins, bApp.BankKeeper.GetAllBalances(migrationCtx, vault), "economy phases must precede module migration")
		require.True(t, bApp.BankKeeper.GetAllBalances(migrationCtx, primary).IsZero())

		if !failModuleMigration {
			return nil
		}
		require.NoError(t, bApp.BankKeeper.SendCoins(migrationCtx, vault, primary, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1))))
		return failure
	}))
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	handler := v22.CreateUpgradeHandler(bApp.ModuleManager, config, delegationUpgradeKeepers(bApp))
	_, err = handler(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
	require.ErrorIs(t, err, failure)
	require.True(t, called)
	assertIndexState(ctx, false)
	require.Equal(t, coins, bApp.BankKeeper.GetAllBalances(ctx, primary))
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, vault).IsZero())
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, vault))
	_, found := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
	require.False(t, found)

	require.Empty(t, ctx.EventManager().Events())

	// Retry the same handler and module versions after removing the injected
	// failure. The rolled-back state must remain a valid upgrade input.
	failModuleMigration = false
	called = false
	_, err = handler(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
	require.NoError(t, err)
	require.True(t, called)
	assertIndexState(ctx, true)

	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, primary).IsZero())
	require.Equal(t, coins, bApp.BankKeeper.GetAllBalances(ctx, vault))
	require.NotNil(t, bApp.AccountKeeper.GetAccount(ctx, vault))
	owner, found := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
	require.True(t, found)
	require.Equal(t, playerID, owner)
}
