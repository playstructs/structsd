package app_test

import (
	"context"
	"errors"
	"math/rand"
	"testing"

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

// Issuance authority and authority over the treasury debit are independent.
// Exercise both through real signatures and ante, including rollback after a
// prior membership edit and after bank collateral/mint writes have begun.
func TestSignedGuildMintTreasuryAuthorityAndRollback(t *testing.T) {
	for _, scenario := range []string{"nonmember_owner", "member_both", "mint_only", "transfer_only", "key_missing_mint", "key_missing_transfer", "after_kick", "final_credit_failure"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			ownerKey, actorKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
			ownerAddress, actorAddress := sdk.AccAddress(ownerKey.PubKey().Address()), sdk.AccAddress(actorKey.PubKey().Address())
			owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, ownerAddress)
			actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, actorAddress)
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-928", 928, owner.Id, owner.Creator
			if scenario == "nonmember_owner" {
				guild.Owner = actor.Id
				require.Empty(t, actor.GuildId)
			} else {
				actor.GuildId, actor.GuildRank = guild.Id, 5
				bApp.StructsKeeper.SetPlayer(ctx, actor)
			}
			bApp.StructsKeeper.SetGuild(ctx, guild)
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			grants := types.PermGuildTokenMint | types.PermTokenTransfer
			if scenario == "mint_only" {
				grants = types.PermGuildTokenMint
			} else if scenario == "transfer_only" {
				grants = types.PermTokenTransfer
			}
			cc.SetPermissionsGuildRank(cc.GetGuild(guild.Id), cc.GetGuild(guild.Id), grants, 5)
			cc.CommitAll()
			if scenario == "key_missing_mint" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(actorAddress.String()), types.PermAll&^types.PermGuildTokenMint)
			} else if scenario == "key_missing_transfer" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(actorAddress.String()), types.PermAll&^types.PermTokenTransfer)
			}
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(guild.Id, actor.Id)
			require.NoError(t, err)
			require.Len(t, vaults, 2)
			treasury, err := types.VaultAddress(guild.Id)
			require.NoError(t, err)
			personal, err := types.VaultAddress(actor.Id)
			require.NoError(t, err)
			fundRedelegationAccount(t, bApp, ctx, treasury, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
			fundRedelegationAccount(t, bApp, ctx, personal, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 50)))
			pool := bApp.StructsKeeper.NewCurrentContext(ctx).GetGuild(guild.Id).GetBankCollateralPool()
			denom := "uguild." + guild.Id
			module := bApp.AccountKeeper.GetModuleAddress(types.ModuleName)
			moduleBefore := bApp.BankKeeper.GetAllBalances(ctx, module)
			alphaSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha")
			creditAttempts := 0
			bApp.BankKeeper.AppendSendRestriction(func(_ context.Context, from, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
				if from.Equals(module) && to.Equals(treasury) && coins.AmountOf(denom).IsPositive() {
					creditAttempts++
					if scenario == "final_credit_failure" {
						return to, errors.New("injected final mint credit refusal")
					}
				}
				return to, nil
			})
			messages := []sdk.Msg{&types.MsgGuildBankMint{Creator: actorAddress.String(), GuildId: guild.Id, AmountAlpha: 60, AmountToken: 10}}
			keys := []cryptotypes.PrivKey{actorKey}
			if scenario == "after_kick" {
				messages = append([]sdk.Msg{&types.MsgGuildMembershipKick{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: actor.Id}}, messages...)
				keys = []cryptotypes.PrivKey{ownerKey, actorKey}
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
			if scenario == "nonmember_owner" || scenario == "member_both" {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, 1, creditAttempts)
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 40), sdk.NewInt64Coin(denom, 10)), bApp.BankKeeper.GetAllBalances(finalCtx, treasury))
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 60)), bApp.BankKeeper.GetAllBalances(finalCtx, pool))
				require.Equal(t, int64(10), bApp.BankKeeper.GetSupply(finalCtx, denom).Amount.Int64())
				mintEvents := 0
				for _, event := range result.Events {
					if event.Type == "structs.structs.EventGuildBankMint" {
						mintEvents++
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						require.Equal(t, &types.EventGuildBankMintDetail{GuildId: guild.Id, AmountAlpha: 60, AmountToken: 10, PlayerId: actor.Id}, decoded.(*types.EventGuildBankMint).EventGuildBankMintDetail)
					}
				}
				require.Equal(t, 1, mintEvents)
			} else {
				require.NotZero(t, result.Code)
				if scenario == "key_missing_mint" || scenario == "key_missing_transfer" {
					require.Contains(t, result.Log, "address lacks required permission bits")
				} else if scenario == "final_credit_failure" {
					require.Contains(t, result.Log, "injected final mint credit refusal")
					require.Equal(t, 1, creditAttempts, "failure must occur after collateral movement and token mint")
				} else {
					require.Contains(t, result.Log, "permission")
					if scenario == "after_kick" {
						require.Contains(t, result.Log, "message index: 1", "kick must execute before mint is refused")
					}
				}
				if scenario != "final_credit_failure" {
					require.Zero(t, creditAttempts, "authorization failure must precede bank issuance")
				}
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)), bApp.BankKeeper.GetAllBalances(finalCtx, treasury))
				require.True(t, bApp.BankKeeper.GetAllBalances(finalCtx, pool).IsZero())
				require.True(t, bApp.BankKeeper.GetSupply(finalCtx, denom).IsZero())
				for _, event := range result.Events {
					require.NotEqual(t, "transfer", event.Type)
					require.NotEqual(t, "coin_received", event.Type)
					require.NotEqual(t, "coin_spent", event.Type)
					require.NotEqual(t, "coinbase", event.Type)
					require.NotEqual(t, "structs.structs.EventGuildBankMint", event.Type)
					require.NotEqual(t, "structs.structs.EventPlayer", event.Type)
				}
			}
			actual, found := bApp.StructsKeeper.GetPlayer(finalCtx, actor.Id)
			require.True(t, found)
			require.Equal(t, actor.GuildId, actual.GuildId, "a refused second message must roll back the kick")
			require.Equal(t, actor.GuildRank, actual.GuildRank)
			require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 50)), bApp.BankKeeper.GetAllBalances(finalCtx, personal), "mint must never use the actor's personal funds")
			require.Equal(t, moduleBefore, bApp.BankKeeper.GetAllBalances(finalCtx, module))
			require.Equal(t, alphaSupply, bApp.BankKeeper.GetSupply(finalCtx, "ualpha"))
			require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(finalCtx, treasury.String()))
			vaultOwner, found := bApp.StructsKeeper.GetVaultOwner(finalCtx, treasury)
			require.True(t, found)
			require.Equal(t, guild.Id, vaultOwner)
		})
	}
}
