package app_test

import (
	"bytes"
	"math/rand"
	"testing"

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

// The live single-validator GP1 phase proves a committed no-op. These signed
// deliveries additionally change the reference to an independently controlled
// reactor, through actual staking lookup, ante permission checks and rollback.
func TestSignedGuildPrimaryReactorAuthorityAndRollback(t *testing.T) {
	for _, scenario := range []string{"owner_secondary", "restricted_owner", "outsider", "member", "granted_admin", "restricted_grantee", "unbonded_target", "jailed_target", "missing_validator", "unknown_reactor", "failed_second_message"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			k := bApp.StructsKeeper
			owner := appendAppPlayer(t, k, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x79}, 20)))
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			actorID := owner.Id
			if scenario == "outsider" || scenario == "member" || scenario == "granted_admin" || scenario == "restricted_grantee" {
				actor := appendAppPlayer(t, k, ctx, address)
				actorID = actor.Id
			} else {
				require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), owner.Index))
			}
			bits := types.PermAdmin
			if scenario == "restricted_owner" || scenario == "restricted_grantee" {
				bits = types.PermUpdate
			}
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			source := createRedelegationValidator(t, bApp, ctx, staking, denom, "guild-primary-source")
			target := createRedelegationValidator(t, bApp, ctx, staking, denom, "guild-primary-target")
			if scenario != "unbonded_target" {
				_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
				require.NoError(t, err)
			}
			if scenario == "jailed_target" {
				validator, err := bApp.StakingKeeper.GetValidator(ctx, target)
				require.NoError(t, err)
				consensus, err := validator.GetConsAddr()
				require.NoError(t, err)
				require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
			}
			validator, err := bApp.StakingKeeper.GetValidator(ctx, target)
			require.NoError(t, err)
			require.Equal(t, scenario != "unbonded_target", validator.IsBonded())


			sourceID, targetID := jailGateReactorId(t, bApp, ctx, source), jailGateReactorId(t, bApp, ctx, target)
			originalTarget, found := k.GetReactor(ctx, targetID)
			require.True(t, found)
			require.NotEqual(t, actorID, originalTarget.Owner)
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-918", 918, owner.Id, owner.Creator
			guild.PrimaryReactorId = sourceID
			require.NotEqual(t, guild.Id, owner.GuildId, "guild ownership does not require membership")
			k.SetGuild(ctx, guild)
			if scenario == "member" {
				actor, found := k.GetPlayer(ctx, actorID)
				require.True(t, found)
				actor.GuildId, actor.GuildRank = guild.Id, guild.EntryRank
				k.SetPlayer(ctx, actor)
			}
			if scenario == "granted_admin" || scenario == "restricted_grantee" {
				k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(guild.Id, actorID), types.PermAdmin)
			}
			if scenario == "unknown_reactor" {
				targetID = "3-999999"
			}
			if scenario == "missing_validator" {
				ghost := k.AppendReactor(ctx, types.Reactor{Validator: sdk.ValAddress(bytes.Repeat([]byte{0x7a}, 20)).String(), RawAddress: bytes.Repeat([]byte{0x7a}, 20)})
				targetID = ghost.Id
			}
			permissions := k.GetAllPermissionExport(ctx)
			messages := []sdk.Msg{&types.MsgGuildUpdatePrimaryReactor{Creator: address.String(), GuildId: guild.Id, ReactorId: targetID}}
			if scenario == "failed_second_message" {
				messages = append(messages, &types.MsgGuildUpdatePrimaryReactor{Creator: address.String(), GuildId: "0-999999", ReactorId: targetID})
			}
			signer := bApp.AccountKeeper.GetAccount(ctx, address)
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
			actual, found := k.GetGuild(finalCtx, guild.Id)
			require.True(t, found)
			allowed := scenario == "owner_secondary" || scenario == "granted_admin" || scenario == "unbonded_target"
			expected := guild
			if allowed {
				require.Zero(t, result.Code, result.Log)
				expected.PrimaryReactorId = targetID
			} else {
				require.NotZero(t, result.Code)
				switch scenario {
				case "failed_second_message":
					require.Contains(t, result.Log, "message index: 1", "first update must succeed before the later refusal")
				case "jailed_target":
					require.Contains(t, result.Log, "jailed")
				case "missing_validator", "unknown_reactor":
					require.Contains(t, result.Log, "not found")
				default:
					require.Contains(t, result.Log, "permission")
				}
			}
			require.Equal(t, expected, actual, "only the intended guild reference may change")
			require.Equal(t, permissions, k.GetAllPermissionExport(finalCtx), "choosing a reactor conveys no permissions")
			afterTarget, found := k.GetReactor(finalCtx, originalTarget.Id)
			require.True(t, found)
			require.Equal(t, originalTarget, afterTarget, "primary selection does not assign reactor ownership or association")
			guildEvents := 0
			for _, event := range result.Events {
				switch event.Type {
				case "structs.structs.EventGuild":
					guildEvents++
					decoded, err := sdk.ParseTypedEvent(event)
					require.NoError(t, err)
					require.NotNil(t, decoded.(*types.EventGuild).Guild)
					require.Equal(t, expected, *decoded.(*types.EventGuild).Guild)
				case "coin_spent", "coin_received", "delegate", "redelegate", "structs.structs.EventInfusion", "structs.structs.EventReactor":
					t.Fatalf("primary reactor selection emitted an asset/ownership event: %s", event.Type)
				}
			}
			if allowed {
				require.Equal(t, 1, guildEvents)
			} else {
				require.Zero(t, guildEvents, "refused or rolled-back selection must not reach indexers")
			}
			cc := k.NewCurrentContext(finalCtx)
			actor, err := cc.GetSigningPlayer(address.String())
			require.NoError(t, err)
			require.Error(t, cc.PermissionCheck(cc.GetReactor(originalTarget.Id), actor, types.PermAdmin), "primary selection conveys no reactor administration")
			requireStakingAndDistributionHealthy(t, bApp, finalCtx)
		})
	}
}
