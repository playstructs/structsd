package app_test

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Issuer burn authority deliberately reaches another holder's vault without
// that holder signing. It must still obey the actor's own key ceiling, current
// rank membership and escrow protection, and roll back an already finished burn
// when a later message fails.
func TestSignedGuildConfiscationAuthorityAndRollback(t *testing.T) {
	for _, scenario := range []string{"nonmember_owner", "rank_burn_only", "missing_key_bit", "outsider", "after_kick", "legacy_escrow", "late_batch_failure"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			ownerKey, actorKey, holderKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
			ownerAddress, actorAddress, holderAddress := sdk.AccAddress(ownerKey.PubKey().Address()), sdk.AccAddress(actorKey.PubKey().Address()), sdk.AccAddress(holderKey.PubKey().Address())
			owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, ownerAddress)
			actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, actorAddress)
			holder := appendAppPlayer(t, bApp.StructsKeeper, ctx, holderAddress)
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner = "0-984", 984, owner.Id
			if scenario == "nonmember_owner" {
				guild.Owner = actor.Id
			} else if scenario != "outsider" {
				actor.GuildId, actor.GuildRank = guild.Id, 5
				bApp.StructsKeeper.SetPlayer(ctx, actor)
			}
			bApp.StructsKeeper.SetGuild(ctx, guild)
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			cc.SetPermissionsGuildRank(cc.GetGuild(guild.Id), cc.GetGuild(guild.Id), types.PermGuildTokenBurn, 5)
			_, err := cc.EnsureVaults(guild.Id, actor.Id, holder.Id)
			require.NoError(t, err)
			cc.CommitAll()
			if scenario == "rank_burn_only" || scenario == "nonmember_owner" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(actorAddress.String()), types.PermGuildTokenBurn)
			} else if scenario == "missing_key_bit" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(actorAddress.String()), types.PermAll&^types.PermGuildTokenBurn)
			}
			personal, err := types.VaultAddress(actor.Id)
			require.NoError(t, err)
			treasury, err := types.VaultAddress(guild.Id)
			require.NoError(t, err)
			victim, err := types.VaultAddress(holder.Id)
			require.NoError(t, err)
			pool := cc.GetGuild(guild.Id).GetBankCollateralPool()
			denom := "uguild." + guild.Id
			module := bApp.AccountKeeper.GetModuleAddress(types.ModuleName)
			fundRedelegationAccount(t, bApp, ctx, personal, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
			fundRedelegationAccount(t, bApp, ctx, victim, sdk.NewCoins(sdk.NewInt64Coin(denom, 500), sdk.NewInt64Coin("ualpha", 200)))
			fundRedelegationAccount(t, bApp, ctx, pool, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1000)))
			if scenario == "legacy_escrow" {
				bApp.StructsKeeper.SetLegacyGuildBankEscrow(ctx, victim.String())
			}
			observed := []sdk.AccAddress{personal, treasury, victim, pool, module, actorAddress, holderAddress}
			before := make([]sdk.Coins, len(observed))
			for i, address := range observed {
				before[i] = bApp.BankKeeper.GetAllBalances(ctx, address)
			}
			alphaSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha")
			lateAttempts := 0
			bApp.BankKeeper.AppendSendRestriction(func(goCtx context.Context, from, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
				if scenario == "late_batch_failure" && from.Equals(personal) && to.Equals(victim) {
					lateAttempts++
					require.Equal(t, math.NewInt(450), bApp.BankKeeper.GetSupply(goCtx, denom).Amount, "first message must have burned before the injected failure")
					require.Equal(t, math.NewInt(450), bApp.BankKeeper.GetBalance(goCtx, victim, denom).Amount)
					return to, errors.New("injected post-confiscation transfer refusal")
				}
				return to, nil
			})
			messages := []sdk.Msg{&types.MsgGuildBankConfiscateAndBurn{Creator: actorAddress.String(), GuildId: guild.Id, Address: victim.String(), AmountToken: 50}}
			keys := []cryptotypes.PrivKey{actorKey}
			if scenario == "after_kick" {
				messages = append([]sdk.Msg{&types.MsgGuildMembershipKick{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: actor.Id}}, messages...)
				keys = []cryptotypes.PrivKey{ownerKey, actorKey}
			} else if scenario == "late_batch_failure" {
				messages = append(messages, &types.MsgVaultSend{Creator: actorAddress.String(), OwnerId: actor.Id, RecipientOwnerId: holder.Id, Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1))})
			}
			var numbers, sequences []uint64
			for _, key := range keys {
				account := bApp.AccountKeeper.GetAccount(ctx, sdk.AccAddress(key.PubKey().Address()))
				numbers = append(numbers, account.GetAccountNumber())
				sequences = append(sequences, account.GetSequence())
			}
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
				ante.DefaultFreeGasCap, jailGateChainID, numbers, sequences, keys...)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			success := scenario == "nonmember_owner" || scenario == "rank_burn_only"
			if success {
				require.Zero(t, result.Code, result.Log)
				before[2] = sdk.NewCoins(sdk.NewInt64Coin("ualpha", 200), sdk.NewInt64Coin(denom, 450))
				require.Equal(t, math.NewInt(450), bApp.BankKeeper.GetSupply(finalCtx, denom).Amount)
				count := 0
				for _, event := range result.Events {
					if event.Type == "structs.structs.EventGuildBankConfiscateAndBurn" {
						count++
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						require.Equal(t, &types.EventGuildBankConfiscateAndBurnDetail{GuildId: guild.Id, AmountToken: 50, Address: victim.String()}, decoded.(*types.EventGuildBankConfiscateAndBurn).EventGuildBankConfiscateAndBurnDetail)
					}
				}
				require.Equal(t, 1, count)
				var bankEvents []map[string]string
				for _, event := range result.Events {
					switch event.Type {
					case "coin_spent", "coin_received", "transfer", "burn", "coinbase":
						attributes := map[string]string{"type": event.Type}
						for _, attribute := range event.Attributes {
							if attribute.Key == "msg_index" {
								require.Equal(t, "0", attribute.Value)
								continue
							}
							require.NotContains(t, attributes, attribute.Key)
							attributes[attribute.Key] = attribute.Value
						}
						bankEvents = append(bankEvents, attributes)
					}
				}
				require.Equal(t, []map[string]string{
					{"type": "coin_spent", "spender": victim.String(), "amount": "50" + denom},
					{"type": "coin_received", "receiver": module.String(), "amount": "50" + denom},
					{"type": "transfer", "sender": victim.String(), "recipient": module.String(), "amount": "50" + denom},
					{"type": "coin_spent", "spender": module.String(), "amount": "50" + denom},
					{"type": "burn", "burner": module.String(), "amount": "50" + denom},
				}, bankEvents, "the transfer and burn are separate bank legs, with no collateral payout")
			} else {
				require.NotZero(t, result.Code)
				require.Equal(t, math.NewInt(500), bApp.BankKeeper.GetSupply(finalCtx, denom).Amount)
				switch scenario {
				case "missing_key_bit":
					require.Contains(t, result.Log, "address lacks required permission bits")
				case "legacy_escrow":
					require.Contains(t, result.Log, "ibc_escrow_backing")
				case "late_batch_failure":
					require.Contains(t, result.Log, "message index: 1")
					require.Contains(t, result.Log, "injected post-confiscation transfer refusal")
				default:
					require.Contains(t, result.Log, "permission")
					if scenario == "after_kick" {
						require.Contains(t, result.Log, "message index: 1")
					}
				}
				for _, event := range result.Events {
					require.False(t, strings.HasPrefix(event.Type, "structs.structs.EventGuildBank"), event.Type)
					require.NotContains(t, []string{"burn", "transfer", "coin_spent", "coin_received", "structs.structs.EventPlayer"}, event.Type)
				}
			}
			if scenario == "late_batch_failure" {
				require.Equal(t, 1, lateAttempts)
			} else {
				require.Zero(t, lateAttempts)
			}
			for i, address := range observed {
				require.Equal(t, before[i], bApp.BankKeeper.GetAllBalances(finalCtx, address), address.String())
			}
			require.Equal(t, alphaSupply, bApp.BankKeeper.GetSupply(finalCtx, "ualpha"))
			actual, found := bApp.StructsKeeper.GetPlayer(finalCtx, actor.Id)
			require.True(t, found)
			require.Equal(t, actor.GuildId, actual.GuildId, "failed burn must restore the prior membership")
			require.Equal(t, actor.GuildRank, actual.GuildRank)
			require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(finalCtx, victim.String()))
			vaultOwner, found := bApp.StructsKeeper.GetVaultOwner(finalCtx, victim)
			require.True(t, found)
			require.Equal(t, holder.Id, vaultOwner)
		})
	}
}
