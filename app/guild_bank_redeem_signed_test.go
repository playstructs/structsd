package app_test

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"strings"
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// The composed conversion deliberately gives its redemption leg no minimum.
// Exercise the public redemption handler's own minimum and permission checks,
// then force a failure after burning to prove real transaction rollback.
func TestSignedGuildBankRedeemAuthorityMinimumAndRollback(t *testing.T) {
	for _, scenario := range []string{"success", "zero_minimum", "slippage", "missing_token_bit", "credit_refused", "second_message_fails"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x6c}, 20)))
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner = "0-983", 983, owner.Id
			guild.BankConvertOutFee = math.LegacyMustNewDecFromStr("0.1")
			bApp.StructsKeeper.SetGuild(ctx, guild)
			require.Empty(t, actor.GuildId, "a token holder does not need guild membership to redeem")
			_, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(actor.Id, owner.Id)
			require.NoError(t, err)
			vault, err := types.VaultAddress(actor.Id)
			require.NoError(t, err)
			ownerVault, err := types.VaultAddress(owner.Id)
			require.NoError(t, err)
			pool := bApp.StructsKeeper.NewCurrentContext(ctx).GetGuild(guild.Id).GetBankCollateralPool()
			denom := "uguild." + guild.Id
			module := bApp.AccountKeeper.GetModuleAddress(types.ModuleName)
			fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
			fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin(denom, 500), sdk.NewInt64Coin("ualpha", 200)))
			fundRedelegationAccount(t, bApp, ctx, pool, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1000)))
			if scenario == "missing_token_bit" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermTokenTransfer)
			}
			observed := []sdk.AccAddress{vault, ownerVault, pool, module, address}
			balances := make([]sdk.Coins, len(observed))
			for i, account := range observed {
				balances[i] = bApp.BankKeeper.GetAllBalances(ctx, account)
			}
			alphaSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha")
			creditAttempts := 0
			bApp.BankKeeper.AppendSendRestriction(func(goCtx context.Context, from, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
				if from.Equals(pool) && to.Equals(vault) && coins.AmountOf("ualpha").IsPositive() {
					creditAttempts++
					require.Equal(t, math.NewInt(450), bApp.BankKeeper.GetSupply(goCtx, denom).Amount, "the source tokens must already be burned at the failure point")
					require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 90)), coins)
					if scenario == "credit_refused" {
						return to, errors.New("injected redemption credit refusal")
					}
				}
				return to, nil
			})
			minimum := uint64(90)
			if scenario == "zero_minimum" {
				minimum = 0
			} else if scenario == "slippage" {
				minimum = 91
			}
			messages := []sdk.Msg{&types.MsgGuildBankRedeem{Creator: address.String(), AmountToken: sdk.NewInt64Coin(denom, 50), MinAmountAlpha: minimum}}
			if scenario == "second_message_fails" {
				messages = append(messages, &types.MsgVaultSend{Creator: address.String(), OwnerId: actor.Id, RecipientOwnerId: owner.Id, Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 291))})
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
			if scenario == "success" {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 290), sdk.NewInt64Coin(denom, 450)), bApp.BankKeeper.GetAllBalances(finalCtx, vault))
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 910)), bApp.BankKeeper.GetAllBalances(finalCtx, pool), "ten alpha stays in collateral as the fee")
				require.Equal(t, math.NewInt(450), bApp.BankKeeper.GetSupply(finalCtx, denom).Amount)
				count := 0
				for _, event := range result.Events {
					if event.Type == "structs.structs.EventGuildBankRedeem" {
						count++
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						require.Equal(t, &types.EventGuildBankRedeemDetail{GuildId: guild.Id, AmountAlpha: 90, AmountToken: 50, PlayerId: actor.Id, Fee: 10}, decoded.(*types.EventGuildBankRedeem).EventGuildBankRedeemDetail)
					}
				}
				require.Equal(t, 1, count)
			} else {
				require.NotZero(t, result.Code)
				for i, account := range observed {
					require.Equal(t, balances[i], bApp.BankKeeper.GetAllBalances(finalCtx, account), account.String())
				}
				require.Equal(t, math.NewInt(500), bApp.BankKeeper.GetSupply(finalCtx, denom).Amount)
				for _, event := range result.Events {
					require.False(t, strings.HasPrefix(event.Type, "structs.structs.EventGuildBank"), event.Type)
					require.NotContains(t, []string{"burn", "coinbase", "transfer", "coin_spent", "coin_received"}, event.Type)
				}
				switch scenario {
				case "zero_minimum":
					require.Contains(t, result.Log, "must_be_positive")
				case "slippage":
					require.Contains(t, result.Log, "slippage")
				case "missing_token_bit":
					require.Contains(t, result.Log, "address lacks required permission bits")
				case "credit_refused":
					require.Contains(t, result.Log, "injected redemption credit refusal")
				case "second_message_fails":
					require.Contains(t, result.Log, "message index: 1")
					require.Contains(t, result.Log, "insufficient funds")
				}
			}
			wantAttempts := 0
			if scenario == "success" || scenario == "credit_refused" || scenario == "second_message_fails" {
				wantAttempts = 1
			}
			require.Equal(t, wantAttempts, creditAttempts)
			require.Equal(t, alphaSupply, bApp.BankKeeper.GetSupply(finalCtx, "ualpha"))
			for _, i := range []int{1, 3, 4} {
				require.Equal(t, balances[i], bApp.BankKeeper.GetAllBalances(finalCtx, observed[i]))
			}
		})
	}
}
