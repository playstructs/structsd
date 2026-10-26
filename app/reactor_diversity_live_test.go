package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func grossForDiversityShare(t *testing.T, share uint64, commission math.LegacyDec) uint64 {
	t.Helper()
	for gross := share; gross <= share*2; gross++ {
		row := types.Infusion{Power: gross, Commission: commission}
		_, _, actual := row.GetPowerDistribution()
		if actual == share {
			return gross
		}
	}
	t.Fatal("fixture commission cannot produce requested share")
	return 0
}

func TestDiversityReadinessSeesNativeStakeAfterFailedInfusionHook(t *testing.T) {
	for _, shape := range []string{"higher-row", "missing-row", "zero-row"} {
		t.Run(shape, func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 21)
			k := bApp.StructsKeeper
			require.True(t, k.ReactorDiversityReady(ctx))
			primary, _, owner := newTransferPlayer(t, bApp, ctx)
			player, found := k.GetPlayer(ctx, owner)
			require.True(t, found)
			// Isolate primary capacity from ordinary passive draw in this test.
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, owner), 0)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			var target sdk.ValAddress
			var reactor types.Reactor
			for i, validator := range validators {
				address, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
				require.NoError(t, err)
				id := jailGateReactorId(t, bApp, ctx, address)
				r, found := k.GetReactor(ctx, id)
				require.True(t, found)
				share := uint64(70)
				if i == 0 {
					share, target, reactor = 600, address, r
				}
				delegate(t, bApp, ctx, server, denom, primary, address, math.NewIntFromUint64(grossForDiversityShare(t, share, r.DefaultCommission)))
			}
			capacityKey := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner)
			require.Equal(t, uint64(2859), k.GetGridAttribute(ctx, capacityKey))
			require.NoError(t, k.CheckReactorDiversityAccounting(ctx))
			destination := primary
			additional := grossForDiversityShare(t, 650, reactor.DefaultCommission) - grossForDiversityShare(t, 600, reactor.DefaultCommission)
			if shape != "higher-row" {
				destination = sdk.AccAddress([]byte("diversity-extra-address-00000001"))
				require.NoError(t, k.SetPlayerIndexForAddress(ctx, destination.String(), player.Index))
				additional = grossForDiversityShare(t, 50, reactor.DefaultCommission)
				if shape == "zero-row" {
					delegate(t, bApp, ctx, server, denom, destination, target, math.NewInt(1))
					_, err := server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(destination.String(), target.String(), sdk.NewInt64Coin(denom, 1)))
					require.NoError(t, err)
				}
			}
			beforeRow, existed := k.GetInfusion(ctx, reactor.Id, destination.String())
			require.Equal(t, shape != "missing-row", existed)
			fundRedelegationAccount(t, bApp, ctx, destination, sdk.NewCoins(sdk.NewCoin(denom, math.NewIntFromUint64(additional))))
			cc := k.NewCurrentContext(ctx)
			require.Equal(t, uint64(2859), cc.GetPlayer(owner).GetAvailableCapacity())
			// Delegate itself does not read UBD rows. Force the Structs hook's
			// auxiliary read to fail, then restore that row to model a recovered
			// backing read while the genuine native delegation remains updated.
			store := ctx.KVStore(bApp.GetKey(stakingtypes.StoreKey))
			key := stakingtypes.GetUBDKey(destination, target)
			previous := append([]byte(nil), store.Get(key)...)
			store.Set(key, []byte{0xff})
			_, err = server.Delegate(ctx, stakingtypes.NewMsgDelegate(destination.String(), target.String(), sdk.NewCoin(denom, math.NewIntFromUint64(additional))))
			if previous == nil {
				store.Delete(key)
			} else {
				store.Set(key, previous)
			}
			require.NoError(t, err, "native stake succeeds while auxiliary reconciliation is deferred")
			afterRow, found := k.GetInfusion(ctx, reactor.Id, destination.String())
			require.Equal(t, existed, found)
			require.Equal(t, beforeRow, afterRow)
			require.Equal(t, uint64(2859), k.GetGridAttribute(ctx, capacityKey), "stored capacity awaits repair")
			for i := 0; i < 2; i++ {
				require.Equal(t, uint64(2805), cc.GetPlayer(owner).GetAvailableCapacity(), "old base 2000 plus current bonus 805, without crediting the new 50")
			}
			eventsBeforeQuery := len(ctx.EventManager().Events())
			query, err := k.OwnerReactorDiversity(ctx, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
			require.NoError(t, err)
			require.True(t, query.LivePortfolioComplete)
			require.True(t, query.InfusionInventoryComplete)
			require.Equal(t, "805", query.LiveBonus)
			require.Equal(t, uint64(859), query.Stored.AppliedBonus)
			require.Equal(t, uint64(805), query.UsableBonus)
			require.Equal(t, uint64(2805), query.UsablePrimaryCapacity)
			require.Equal(t, uint64(2859), query.StoredCapacity)
			var roundTrip types.QueryOwnerReactorDiversityResponse
			require.NoError(t, bApp.AppCodec().UnmarshalJSON(bApp.AppCodec().MustMarshalJSON(query), &roundTrip))
			require.Equal(t, *query, roundTrip)
			require.Len(t, ctx.EventManager().Events(), eventsBeforeQuery)
			require.NotEmpty(t, k.GetReactorSlashReconcileQueue(ctx))
			k.ProcessReactorSlashReconcileQueue(ctx)
			require.Equal(t, uint64(2855), k.GetGridAttribute(ctx, capacityKey))
			require.Equal(t, uint64(2855), k.NewCurrentContext(ctx).GetPlayer(owner).GetAvailableCapacity())
			require.NoError(t, k.CheckReactorDiversityAccounting(ctx))
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
}
