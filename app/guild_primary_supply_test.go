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

func TestGuildAllocationUsesLiveVaultInfusionBacking(t *testing.T) {
	for _, jailed := range []bool{false, true} {
		name := "healthy"
		if jailed {
			name = "jailed"
		}
		t.Run(name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			primary, _, ownerID := newTransferPlayer(t, bApp, ctx)
			_, _, consumerID := newTransferPlayer(t, bApp, ctx)
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner = "0-99401", 99401, ownerID
			bApp.StructsKeeper.SetGuild(ctx, guild)
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			cc.GetPlayer(consumerID).GetFleet().GetCommandStruct()
			vaults, err := cc.EnsureVaults(guild.Id)
			require.NoError(t, err)
			cc.CommitAll()


			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, vaults[0], validator, math.NewInt(3000000))
			reactorID := jailGateReactorId(t, bApp, ctx, validator)
			row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vaults[0].String())
			require.True(t, found)
			require.Equal(t, guild.Id, row.AssetOwnerID())
			const stationID = "4-99401"
			bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: stationID, Owner: guild.Id, Creator: primary.String()})
			server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			created, err := server.AllocationCreate(ctx, &types.MsgAllocationCreate{Creator: primary.String(), Controller: guild.Id, SourceObjectId: guild.Id, AllocationType: types.AllocationType_dynamic, Power: 100000})
			require.NoError(t, err)
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			allocation, found := cc.GetAllocation(created.AllocationId)
			require.True(t, found)
			require.NoError(t, allocation.SetDestination(stationID))
			cc.GetPlayer(consumerID).MigrateSubstation(stationID)
			cc.CommitAll()
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			require.True(t, cc.GetPlayer(consumerID).IsOnline())
			if jailed {
				consensus, err := validators[0].GetConsAddr()
				require.NoError(t, err)
				require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
			}
			beforeGrid := bApp.StructsKeeper.GetAllGridExport(ctx)
			beforeInfusions := bApp.StructsKeeper.GetAllInfusion(ctx)
			beforeCount := bApp.StructsKeeper.GetAllocationCount(ctx)
			request := &types.MsgAllocationCreate{Creator: primary.String(), Controller: guild.Id, SourceObjectId: guild.Id, AllocationType: types.AllocationType_dynamic, Power: 1}
			_, err = server.AllocationCreate(ctx, request)
			if !jailed {
				require.NoError(t, err)
				return
			}
			require.Error(t, err, "guild ownership must not bypass the player source's live backing gate")
			require.Equal(t, beforeGrid, bApp.StructsKeeper.GetAllGridExport(ctx))
			require.Equal(t, beforeCount, bApp.StructsKeeper.GetAllocationCount(ctx))
			// Secondary capacity follows the bounded cascade, while direct source admission is gated immediately.
			require.True(t, cc.GetPlayer(consumerID).IsOnline())
			require.Equal(t, uint64(90000), cc.GetSubstation(stationID).GetAvailableCapacity())
			allocation, found = cc.GetAllocation(created.AllocationId)
			require.True(t, found)
			_, err = allocation.SetDynamicPower(100001)
			require.Error(t, err)
			_, err = allocation.SetDynamicPower(99999)
			require.NoError(t, err)
			require.NoError(t, allocation.Destroy())
			cc.CommitAll()
			request.AllocationType, request.Power = types.AllocationType_automated, 0
			_, err = server.AllocationCreate(ctx, request)
			require.ErrorContains(t, err, "unsupported_supply")
			require.Equal(t, beforeCount, bApp.StructsKeeper.GetAllocationCount(ctx))
			require.Equal(t, beforeInfusions, bApp.StructsKeeper.GetAllInfusion(ctx))
		})
	}
}
