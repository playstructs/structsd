package app_test

import (
	"testing"

	"cosmossdk.io/math"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Generator fuel is burned rather than staked, but stored capacity must still
// follow the generator's lifecycle. Exercise the actual infusion/deactivation
// handlers without erasing the raw accounting or already-burned fuel.
func TestPlayerSupplyExcludesOfflineGenerator(t *testing.T) {
	for _, transition := range []string{"healthy", "deactivate", "cached_offline", "cached_destroyed", "missing", "invalid_generation", "independent_reactor"} {
		t.Run(transition, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			primary, _, ownerID := newTransferPlayer(t, bApp, ctx)
			_, _, consumerID := newTransferPlayer(t, bApp, ctx)
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			cc.GetPlayer(ownerID).GetFleet().GetCommandStruct()
			cc.GetPlayer(consumerID).GetFleet().GetCommandStruct()
			cc.CommitAll()
			generatorType, found := bApp.StructsKeeper.GetStructType(ctx, 20)
			require.True(t, found)
			const generatorID, planetID, stationID = "5-99301", "2-99301", "4-99301"
			bApp.StructsKeeper.SetPlanet(ctx, types.Planet{Id: planetID, Owner: ownerID, Creator: primary.String()})
			bApp.StructsKeeper.SetStruct(ctx, types.Struct{Id: generatorID, Owner: ownerID, Creator: primary.String(), Type: 20, LocationId: planetID, LocationType: types.ObjectType_planet})
			statusID := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, generatorID)
			bApp.StructsKeeper.SetStructAttribute(ctx, statusID, uint64(types.StructStateMaterialized|types.StructStateBuilt|types.StructStateOnline))
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			cc.GetPlayer(ownerID).StructsLoadIncrement(generatorType.PassiveDraw)
			vaults, err := cc.EnsureVaults(ownerID)
			require.NoError(t, err)
			cc.CommitAll()
			fundRedelegationAccount(t, bApp, ctx, vaults[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", 2000000)))
			server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			_, err = server.StructGeneratorInfuse(ctx, &types.MsgStructGeneratorInfuse{Creator: primary.String(), StructId: generatorID, InfuseAmount: "2000000ualpha"})
			require.NoError(t, err)


			if transition == "independent_reactor" {
				validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
				require.NoError(t, err)
				validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
				require.NoError(t, err)
				denom, err := bApp.StakingKeeper.BondDenom(ctx)
				require.NoError(t, err)
				delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(3000000))
			}
			bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: stationID, Owner: ownerID, Creator: primary.String()})
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			feeding, err := cc.NewAllocation(types.AllocationType_dynamic, ownerID, stationID, primary.String(), ownerID, 100000)
			require.NoError(t, err)
			cc.GetPlayer(consumerID).MigrateSubstation(stationID)
			provider := cc.NewProvider(types.Provider{
				Owner: ownerID, Creator: primary.String(), SubstationId: stationID,
				Rate: sdk.NewInt64Coin("ualpha", 1), AccessPolicy: types.ProviderAccessPolicy_openMarket,
				CapacityMinimum: 1, CapacityMaximum: 100000, DurationMinimum: 1, DurationMaximum: 10,
				ProviderCancellationPenalty: math.LegacyZeroDec(), ConsumerCancellationPenalty: math.LegacyZeroDec(),
			})
			consumerVaults, err := cc.EnsureVaults(consumerID)
			require.NoError(t, err)
			consumer := cc.GetPlayer(consumerID).GetPlayer()
			cc.CommitAll()
			fundRedelegationAccount(t, bApp, ctx, consumerVaults[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", 10)))
			purchase := &types.MsgAgreementOpen{Creator: consumer.Creator, ProviderId: provider.ID(), Capacity: 1, Duration: 5}
			healthy, _ := ctx.CacheContext()
			_, err = server.AgreementOpen(healthy, purchase)
			require.NoError(t, err, "the exact paid request succeeds before power loss")
			require.Equal(t, int64(5), bApp.BankKeeper.GetBalance(healthy, provider.GetCollateralPoolLocation(), "ualpha").Amount.Int64())
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			require.True(t, cc.GetPlayer(ownerID).IsOnline())
			require.True(t, cc.GetPlayer(consumerID).IsOnline())
			beforeCapacity := cc.GetPlayer(ownerID).GetCapacity()
			beforeInfusions := bApp.StructsKeeper.GetAllInfusion(ctx)
			beforeSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount
			if transition == "deactivate" || transition == "independent_reactor" {
				_, err = server.StructDeactivate(ctx, &types.MsgStructDeactivate{Creator: primary.String(), StructId: generatorID})
				require.NoError(t, err)
				cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			} else if transition == "cached_offline" {
				cc.GetStruct(generatorID).GoOffline()
			} else if transition == "missing" {
				bApp.StructsKeeper.RemoveStruct(ctx, generatorID)
				cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			} else if transition == "invalid_generation" {
				generatorType.PowerGeneration = types.TechPowerGeneration(500)
				bApp.StructsKeeper.SetStructType(ctx, generatorType)
				cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			} else if transition == "cached_destroyed" {
				// A destroyed row still has its online bit. Existence or online
				// alone must not authorize power, including before cache commit.
				cc.GetStruct(generatorID).StatusAddDestroyed()
			}
			owner := cc.GetPlayer(ownerID)
			require.Equal(t, beforeCapacity, owner.GetCapacity())
			if transition == "independent_reactor" {
				require.True(t, owner.IsOnline())
				require.True(t, cc.GetPlayer(consumerID).IsOnline())
				expectedHeadroom := beforeCapacity - 4000000 - owner.GetLoad() - owner.GetStructsLoad()
				require.Equal(t, expectedHeadroom, owner.GetAvailableCapacity())
				_, err = cc.NewAllocation(types.AllocationType_dynamic, ownerID, "", primary.String(), ownerID, expectedHeadroom)
				require.NoError(t, err, "independent power remains allocatable")
				require.Equal(t, beforeInfusions, bApp.StructsKeeper.GetAllInfusion(ctx))
				require.Equal(t, beforeSupply, bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount)
				return
			}
			if transition == "healthy" {
				require.True(t, owner.IsOnline())
				require.True(t, cc.GetPlayer(consumerID).IsOnline())
				_, err = cc.NewAllocation(types.AllocationType_dynamic, ownerID, "", primary.String(), ownerID, 1)
				require.NoError(t, err)
				return
			}
			require.False(t, owner.IsOnline(), "an offline generator cannot keep its owner powered")
			require.Zero(t, owner.GetAvailableCapacity())
			require.True(t, cc.GetPlayer(consumerID).IsOnline(), "downstream service follows queued grid accounting")
			beforeCount := bApp.StructsKeeper.GetAllocationCount(ctx)
			_, err = cc.NewAllocation(types.AllocationType_dynamic, ownerID, "", primary.String(), ownerID, 1)
			require.Error(t, err)
            cc.CommitAll()
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			owner = cc.GetPlayer(ownerID)
			allocation, found := cc.GetAllocation(feeding.ID())
			require.True(t, found)
			_, err = allocation.SetDynamicPower(100001)
			require.Error(t, err)
			_, err = allocation.SetDynamicPower(99999)
			require.NoError(t, err, "reducing load during cleanup remains possible")
			require.NoError(t, allocation.Destroy())
			require.Zero(t, owner.GetLoad(), "avoid an incidental automated-conflict refusal")
			_, err = cc.NewAllocation(types.AllocationType_automated, ownerID, "", primary.String(), ownerID, 0)
			require.ErrorContains(t, err, "unsupported_supply")
			cc.CommitAll()
			require.Equal(t, beforeCount, bApp.StructsKeeper.GetAllocationCount(ctx))
			require.Equal(t, beforeInfusions, bApp.StructsKeeper.GetAllInfusion(ctx), "readiness does not erase burned fuel or settle its stored contribution")
			require.Equal(t, beforeSupply, bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount)
		})
	}
}
