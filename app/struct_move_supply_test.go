package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestStructMoveRequiresLiveOwnerSupplyBeforeDeferredRepair(t *testing.T) {
	for _, mode := range []string{"healthy", "jailed", "jailed_with_backup"} {
		t.Run(mode, func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 2)
			ctx = ctx.WithBlockHeight(100)
			address, _, playerID := newTransferPlayer(t, bApp, ctx)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, address, validator, math.NewInt(10000000))
			if mode == "jailed_with_backup" {
				backup, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
				require.NoError(t, err)
				delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, address, backup, math.NewInt(10000000))
			}
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			command := cc.GetPlayer(playerID).GetFleet().GetCommandStruct()
			commandID := command.ID()
			cc.CommitAll()

			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			owner := cc.GetPlayer(playerID)
			rawCapacity := owner.GetCapacity()
			require.True(t, owner.IsOnline())
			require.Positive(t, owner.GetStructsLoad())
			if mode != "healthy" {
				value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
				require.NoError(t, err)
				consensus, err := value.GetConsAddr()
				require.NoError(t, err)
				require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
			}
			require.Equal(t, rawCapacity, owner.GetCapacity(), "deferred raw power has not been repaired")
			require.Equal(t, mode != "jailed", owner.IsOnline(), "live staking state overrides stored capacity")
			before, found := bApp.StructsKeeper.GetStruct(ctx, commandID)
			require.True(t, found)
			require.NotEqual(t, types.Ambit_space, before.OperatingAmbit)
			beforeGrid := bApp.StructsKeeper.GetAllGridExport(ctx)
			beforeInfusions := bApp.StructsKeeper.GetAllInfusion(ctx)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).StructMove(ctx, &types.MsgStructMove{
				Creator: address.String(), StructId: commandID, LocationType: types.ObjectType_fleet, Ambit: types.Ambit_space,
			})
			after, found := bApp.StructsKeeper.GetStruct(ctx, commandID)
			require.True(t, found)
			if mode == "jailed" {
				require.Error(t, err)
				var powerError *types.PlayerPowerError
				require.ErrorAs(t, err, &powerError)
				require.Equal(t, "offline", powerError.Reason)
				require.Equal(t, before, after)
				require.Equal(t, beforeGrid, bApp.StructsKeeper.GetAllGridExport(ctx))
				for _, event := range ctx.EventManager().Events() {
					require.NotEqual(t, "structs.structs.EventStruct", event.Type)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, types.Ambit_space, after.OperatingAmbit)
			}
			require.Equal(t, beforeInfusions, bApp.StructsKeeper.GetAllInfusion(ctx), "readiness does not consume the repair queue")
		})
	}
}
