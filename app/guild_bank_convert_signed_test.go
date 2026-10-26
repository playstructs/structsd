package app_test

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Exercise the actual ante/message caches, not a manually discarded context.
// A composed conversion can burn source tokens before the destination refuses;
// no source burn, destination mint, bridge transfer or economic event may survive.
func TestSignedGuildBankConversionBoundariesAndRollback(t *testing.T) {
	for _, scenario := range []string{"success", "fees", "supply_boundary", "supply_overflow", "direct_supply_overflow", "slippage", "credit_refused", "second_message_fails", "missing_token_bit", "vesting_locked"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x6b}, 20)))
			source, target := types.CreateEmptyGuild(), types.CreateEmptyGuild()
			source.Id, source.Index, source.Owner = "0-981", 981, owner.Id
			target.Id, target.Index, target.Owner = "0-982", 982, owner.Id
			if scenario == "fees" {
				source.BankConvertOutFee = math.LegacyMustNewDecFromStr("0.1")
				target.BankConvertInFee = math.LegacyMustNewDecFromStr("0.1")
			}
			bApp.StructsKeeper.SetGuild(ctx, source)
			bApp.StructsKeeper.SetGuild(ctx, target)
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(actor.Id, owner.Id)
			require.NoError(t, err)
			vault, err := types.VaultAddress(actor.Id)
			require.NoError(t, err)
			ownerVault, err := types.VaultAddress(owner.Id)
			require.NoError(t, err)
			require.Len(t, vaults, 2)
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			sourcePool := cc.GetGuild(source.Id).GetBankCollateralPool()
			targetPool := cc.GetGuild(target.Id).GetBankCollateralPool()
			sourceDenom, targetDenom := "uguild."+source.Id, "uguild."+target.Id
			targetSupply, targetCollateral := math.NewInt(500), math.NewInt(1000)
			max := math.NewIntFromUint64(^uint64(0))
			if scenario == "supply_boundary" {
				targetSupply, targetCollateral = max.SubRaw(100), max.SubRaw(100)
			} else if strings.HasSuffix(scenario, "supply_overflow") {
				targetSupply = max.SubRaw(50)
			}
			fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
			fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin(sourceDenom, 500), sdk.NewInt64Coin("ualpha", 200)))
			fundRedelegationAccount(t, bApp, ctx, ownerVault, sdk.NewCoins(sdk.NewCoin(targetDenom, targetSupply)))
			fundRedelegationAccount(t, bApp, ctx, sourcePool, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1000)))
			fundRedelegationAccount(t, bApp, ctx, targetPool, sdk.NewCoins(sdk.NewCoin("ualpha", targetCollateral)))
			if scenario == "missing_token_bit" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermTokenTransfer)
			}
			var lockedAccount *types.VaultVestingAccount
			if scenario == "vesting_locked" {
				schedule, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccountWithAddress(address), sdk.NewCoins(sdk.NewInt64Coin(sourceDenom, 500)), ctx.BlockTime().Add(time.Hour).Unix())
				require.NoError(t, err)
				lockedAccount, err = types.NewVaultVestingAccount(actor.Id, bApp.AccountKeeper.GetAccount(ctx, vault).(*authtypes.BaseAccount), nil, schedule)
				require.NoError(t, err)
				bApp.AccountKeeper.SetAccount(ctx, lockedAccount)
				require.True(t, bApp.BankKeeper.SpendableCoin(ctx, vault, sourceDenom).IsZero())
			}
			module := bApp.AccountKeeper.GetModuleAddress(types.ModuleName)
			observed := []sdk.AccAddress{vault, ownerVault, sourcePool, targetPool, module}
			balances := make([]sdk.Coins, len(observed))
			for i, account := range observed {
				balances[i] = bApp.BankKeeper.GetAllBalances(ctx, account)
			}
			alphaSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha")
			bridgeAttempts, creditAttempts := 0, 0
			bApp.BankKeeper.AppendSendRestriction(func(_ context.Context, from, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
				if from.Equals(sourcePool) && to.Equals(vault) && coins.AmountOf("ualpha").IsPositive() {
					bridgeAttempts++
				}
				if to.Equals(vault) && coins.AmountOf(targetDenom).IsPositive() {
					creditAttempts++
					if scenario == "credit_refused" {
						return to, errors.New("injected conversion credit refusal")
					}
				}
				return to, nil
			})
			minimum := uint64(1)
			if scenario == "slippage" {
				minimum = 1000000
			}
			messages := []sdk.Msg{&types.MsgGuildBankConvertToken{Creator: address.String(), AmountToken: sdk.NewInt64Coin(sourceDenom, 50), GuildId: target.Id, MinAmountToken: minimum}}
			if scenario == "direct_supply_overflow" {
				messages = []sdk.Msg{&types.MsgGuildBankConvert{Creator: address.String(), GuildId: target.Id, AmountAlpha: 100, MinAmountToken: 1}}
			}
			if scenario == "second_message_fails" {
				messages = append(messages, &types.MsgVaultSend{Creator: address.String(), OwnerId: actor.Id, RecipientOwnerId: owner.Id, Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 201))})
			}
			signer := bApp.AccountKeeper.GetAccount(ctx, address)
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages,
				sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1)), ante.DefaultFreeGasCap, jailGateChainID,
				[]uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			succeeds := scenario == "success" || scenario == "fees" || scenario == "supply_boundary"
			if succeeds {
				require.Zero(t, result.Code, result.Log)
				bridge, output := int64(100), int64(50)
				if scenario == "fees" {
					bridge, output = 90, 40
				} else if scenario == "supply_boundary" {
					output = 100
				}
				require.Equal(t, math.NewInt(450), bApp.BankKeeper.GetBalance(finalCtx, vault, sourceDenom).Amount)
				require.Equal(t, math.NewInt(output), bApp.BankKeeper.GetBalance(finalCtx, vault, targetDenom).Amount)
				require.Equal(t, math.NewInt(200), bApp.BankKeeper.GetBalance(finalCtx, vault, "ualpha").Amount)
				require.Equal(t, math.NewInt(1000-bridge), bApp.BankKeeper.GetBalance(finalCtx, sourcePool, "ualpha").Amount)
				require.Equal(t, targetCollateral.AddRaw(bridge), bApp.BankKeeper.GetBalance(finalCtx, targetPool, "ualpha").Amount)
				require.Equal(t, math.NewInt(450), bApp.BankKeeper.GetSupply(finalCtx, sourceDenom).Amount)
				require.Equal(t, targetSupply.AddRaw(output), bApp.BankKeeper.GetSupply(finalCtx, targetDenom).Amount)
				require.Equal(t, balances[1], bApp.BankKeeper.GetAllBalances(finalCtx, ownerVault))
				require.Equal(t, balances[4], bApp.BankKeeper.GetAllBalances(finalCtx, module))
				counts := map[string]int{}
				for _, event := range result.Events {
					counts[event.Type]++
					if event.Type == "structs.structs.EventGuildBankConvertToken" {
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						detail := decoded.(*types.EventGuildBankConvertToken).EventGuildBankConvertTokenDetail
						require.Equal(t, &types.EventGuildBankConvertTokenDetail{SourceGuildId: source.Id, TargetGuildId: target.Id, AmountTokenIn: 50, BridgeAlpha: uint64(bridge), AmountTokenOut: uint64(output), PlayerId: actor.Id}, detail)
					}
				}
				for _, name := range []string{"EventGuildBankRedeem", "EventGuildBankConvert", "EventGuildBankConvertToken"} {
					require.Equal(t, 1, counts["structs.structs."+name], name)
				}
			} else {
				require.NotZero(t, result.Code, result.Log)
				for i, account := range observed {
					require.Equal(t, balances[i], bApp.BankKeeper.GetAllBalances(finalCtx, account), account.String())
				}
				require.Equal(t, math.NewInt(500), bApp.BankKeeper.GetSupply(finalCtx, sourceDenom).Amount)
				require.Equal(t, targetSupply, bApp.BankKeeper.GetSupply(finalCtx, targetDenom).Amount)
				for _, event := range result.Events {
					require.False(t, strings.HasPrefix(event.Type, "structs.structs.EventGuildBank"), event.Type)
					require.NotContains(t, []string{"burn", "coinbase", "structs.structs.EventVaultAddress"}, event.Type)
				}
				switch scenario {
				case "supply_overflow", "direct_supply_overflow":
					require.Contains(t, result.Log, "bankSupply")
					require.Contains(t, result.Log, "out_of_range")
				case "slippage":
					require.Contains(t, result.Log, "slippage")
				case "credit_refused":
					require.Contains(t, result.Log, "injected conversion credit refusal")
				case "second_message_fails", "vesting_locked":
					require.Contains(t, result.Log, "insufficient funds")
				case "missing_token_bit":
					require.Contains(t, result.Log, "permission")
				}
			}
			wantBridge, wantCredit := 1, 0
			if scenario == "missing_token_bit" || scenario == "vesting_locked" || scenario == "direct_supply_overflow" {
				wantBridge = 0
			}
			if succeeds || scenario == "credit_refused" || scenario == "second_message_fails" {
				wantCredit = 1
			}
			require.Equal(t, wantBridge, bridgeAttempts, "prove the first leg ran before a later refusal")
			require.Equal(t, wantCredit, creditAttempts, "prove whether the destination mint reached its credit")
			require.Equal(t, alphaSupply, bApp.BankKeeper.GetSupply(finalCtx, "ualpha"))
			require.Equal(t, math.NewInt(100), bApp.BankKeeper.GetBalance(finalCtx, address, "ualpha").Amount, "free conversions leave the signing account untouched")
			require.True(t, bApp.BankKeeper.GetBalance(finalCtx, address, sourceDenom).IsZero())
			require.True(t, bApp.BankKeeper.GetBalance(finalCtx, address, targetDenom).IsZero())
			if lockedAccount != nil {
				require.Equal(t, lockedAccount, bApp.AccountKeeper.GetAccount(finalCtx, vault))
			}
		})
	}
}
