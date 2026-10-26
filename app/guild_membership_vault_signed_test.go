package app_test

import (
	"bytes"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSignedGuildJoinVaultAuthorityAndRollback(t *testing.T) {
	for _, scenario := range []string{"restricted_key", "membership_only_without_move", "authorized_move", "failed_second_message", "delegated_membership_only", "delegated_migration_authority"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			primary := sdk.AccAddress(bytes.Repeat([]byte{0x76}, 20))
			player := appendAppPlayer(t, bApp.StructsKeeper, ctx, primary)
			key := secp256k1.GenPrivKey()
			actor := sdk.AccAddress(key.PubKey().Address())
			bits := types.PermGuildMembership
			if scenario == "authorized_move" || scenario == "failed_second_message" {
				bits |= types.PermTokenMigrate
			}
			if scenario == "delegated_membership_only" || scenario == "delegated_migration_authority" {
				other := appendAppPlayer(t, bApp.StructsKeeper, ctx, actor)
				grant := types.PermGuildMembership
				if scenario == "delegated_migration_authority" {
					grant |= types.PermTokenMigrate
				}
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(player.Id, other.Id), grant)
				// This key has both bits; only its grant over the asset owner varies.
				bits |= types.PermTokenMigrate
			} else {
				require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, actor.String(), player.Index))
			}
			bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(actor.String()), bits)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			source := createRedelegationValidator(t, bApp, ctx, staking, denom, "signed-join-source")
			destination := createRedelegationValidator(t, bApp, ctx, staking, denom, "signed-join-destination")
			_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
			require.NoError(t, err)


			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(player.Id)
			require.NoError(t, err)
			vault := vaults[0]
			stake := math.NewInt(1000000)
			delegate(t, bApp, ctx, staking, denom, vault, source, stake)
			sourceID := jailGateReactorId(t, bApp, ctx, source)
			destinationID := jailGateReactorId(t, bApp, ctx, destination)
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-915", 915, player.Id, primary.String()
			guild.PrimaryReactorId, guild.JoinInfusionMinimum = destinationID, stake.Uint64()
			bApp.StructsKeeper.SetGuild(ctx, guild)
			if scenario == "membership_only_without_move" {
				reactor, found := bApp.StructsKeeper.GetReactor(ctx, sourceID)
				require.True(t, found)
				reactor.GuildId = guild.Id
				bApp.StructsKeeper.SetReactor(ctx, reactor)
			}
			before, found := bApp.StructsKeeper.GetInfusion(ctx, sourceID, vault.String())
			require.True(t, found)
			beforeStake, err := bApp.StakingKeeper.GetDelegation(ctx, vault, source)
			require.NoError(t, err)
			capacity := redelegationCapacity(t, bApp, ctx, player.Id)
			messages := []sdk.Msg{&types.MsgGuildMembershipJoin{
				Creator: actor.String(), PlayerId: player.Id, GuildId: guild.Id,
				InfusionId: []string{sourceID + "-" + vault.String()},
			}}
			if scenario == "failed_second_message" {
				messages = append(messages, &types.MsgGuildMembershipJoin{
					Creator: actor.String(), PlayerId: player.Id, GuildId: "0-999999",
				})
			}
			signer := bApp.AccountKeeper.GetAccount(ctx, actor)
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
				ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			actual, found := bApp.StructsKeeper.GetPlayer(finalCtx, player.Id)
			require.True(t, found)
			failed := scenario == "restricted_key" || scenario == "failed_second_message" || scenario == "delegated_membership_only"
			if failed {
				require.NotZero(t, result.Code)
				if scenario != "failed_second_message" {
					require.Contains(t, result.Log, "permission")
				} else {
					require.Contains(t, result.Log, "message index: 1", "the first join must complete before the batch fails")
				}
				require.Equal(t, player.GuildId, actual.GuildId)
				require.Equal(t, player.GuildRank, actual.GuildRank)
				for _, event := range result.Events {
					require.NotEqual(t, "redelegate", event.Type)
					require.NotEqual(t, "structs.structs.EventInfusion", event.Type)
					require.NotEqual(t, "structs.structs.EventGuildMembershipApplication", event.Type)
				}
			} else {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, guild.Id, actual.GuildId)
				require.Equal(t, guild.EntryRank, actual.GuildRank)
			}
			if scenario != "authorized_move" && scenario != "delegated_migration_authority" {
				restored, found := bApp.StructsKeeper.GetInfusion(finalCtx, sourceID, vault.String())
				require.True(t, found)
				require.Equal(t, before, restored)
				restoredStake, err := bApp.StakingKeeper.GetDelegation(finalCtx, vault, source)
				require.NoError(t, err)
				require.Equal(t, beforeStake, restoredStake)
				_, err = bApp.StakingKeeper.GetDelegation(finalCtx, vault, destination)
				require.Error(t, err)
				_, err = bApp.StakingKeeper.GetRedelegation(finalCtx, vault, source, destination)
				require.Error(t, err)
				require.Equal(t, capacity, redelegationCapacity(t, bApp, finalCtx, player.Id))
			} else {
				_, err = bApp.StakingKeeper.GetDelegation(finalCtx, vault, source)
				require.Error(t, err)
				infusion, found := bApp.StructsKeeper.GetInfusion(finalCtx, destinationID, vault.String())
				require.True(t, found)
				require.Equal(t, stake.Uint64(), infusion.Fuel)
				require.Equal(t, player.Id, infusion.PlayerId)
				_, _, power := infusion.GetPowerDistribution()
				require.Equal(t, power, redelegationCapacity(t, bApp, finalCtx, player.Id))
				_, err = bApp.StakingKeeper.GetRedelegation(finalCtx, vault, source, destination)
				require.NoError(t, err)
				var redelegationEvent, membershipEvent bool
				for _, event := range result.Events {
					redelegationEvent = redelegationEvent || event.Type == "redelegate"
					membershipEvent = membershipEvent || event.Type == "structs.structs.EventGuildMembershipApplication"
				}
				require.True(t, redelegationEvent)
				require.True(t, membershipEvent)
			}
			require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(finalCtx, vault.String()))
			requireStakingAndDistributionHealthy(t, bApp, finalCtx)
		})
	}
}
