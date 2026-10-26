package app_test

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

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

func TestSignedPaidGuildCreationAuthorityAndRollback(t *testing.T) {
	for _, scenario := range []string{"success", "missing_token_bit", "missing_play_bit", "credit_refused", "second_message_fails", "underfunded", "vesting_locked", "vesting_partial_success", "collateral_dust", "jailed_reactor"} {
		t.Run(scenario, func(t *testing.T) {
			succeeds := scenario == "success" || scenario == "vesting_partial_success" || scenario == "collateral_dust"
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			founder := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
   require.NoError(t, err)
   validatorAddress, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
   require.NoError(t, err)
   reactor, found := bApp.StructsKeeper.GetReactor(ctx, jailGateReactorId(t, bApp, ctx, validatorAddress))
   require.True(t, found)
   if scenario == "jailed_reactor" { consAddr, err := validators[0].GetConsAddr(); require.NoError(t, err); require.NoError(t, bApp.StakingKeeper.Jail(ctx, consAddr)) }
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(founder.Id)
			require.NoError(t, err)
			initialVault, initialKey := int64(2000000), int64(100)
			if scenario == "underfunded" {
				initialVault, initialKey = 1999999, 2000100
			} else if scenario == "vesting_partial_success" {
				initialVault = 3000000
			}
			fundRedelegationAccount(t, bApp, ctx, vaults[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", initialVault)))
			fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin("ualpha", initialKey)))
			var vestingAccount *types.VaultVestingAccount
			if scenario == "vesting_locked" || scenario == "vesting_partial_success" {
				locked := int64(2000000)
				if succeeds {
					locked = 1000000
				}
				schedule, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccountWithAddress(address), sdk.NewCoins(sdk.NewInt64Coin("ualpha", locked)), ctx.BlockTime().Add(time.Hour).Unix())
				require.NoError(t, err)
				vestingAccount, err = types.NewVaultVestingAccount(founder.Id, bApp.AccountKeeper.GetAccount(ctx, vaults[0]).(*authtypes.BaseAccount), nil, schedule)
				require.NoError(t, err)
				bApp.AccountKeeper.SetAccount(ctx, vestingAccount)
				require.Equal(t, initialVault-locked, bApp.BankKeeper.SpendableCoin(ctx, vaults[0], "ualpha").Amount.Int64())
			}
			if scenario == "missing_token_bit" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermTokenTransfer)
			}
			if scenario == "missing_play_bit" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermPlay)
			}
			count := bApp.StructsKeeper.GetGuildCount(ctx)
			guildID := keeperlib.GetObjectID(types.ObjectType_guild, count)
			treasury, err := types.VaultAddress(guildID)
			require.NoError(t, err)
			denom := "uguild." + guildID
   if scenario == "collateral_dust" {
    pool := authtypes.NewModuleAddress(types.GuildBankCollateralPool + guildID)
    fundRedelegationAccount(t, bApp, ctx, pool, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1)))
   }
			supply := bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount
			permissions := bApp.StructsKeeper.GetAllPermissionExport(ctx)
			refusals, creditsAttempted := 0, 0
			bApp.BankKeeper.AppendSendRestriction(func(_ context.Context, _, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
				if to.Equals(treasury) && coins.AmountOf(denom).IsPositive() {
					creditsAttempted++
				}
				if scenario == "credit_refused" && to.Equals(treasury) && coins.AmountOf(denom).IsPositive() {
					refusals++
					return to, errors.New("injected signed guild credit refusal")
				}
				return to, nil
			})
			messages := []sdk.Msg{&types.MsgGuildCreate{Creator: address.String(), ReactorId: reactor.Id}}
			if scenario == "second_message_fails" {
				messages = append(messages, &types.MsgVaultSend{Creator: address.String(), OwnerId: founder.Id, RecipientOwnerId: founder.Id, Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1))})
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
			guild, found := bApp.StructsKeeper.GetGuild(finalCtx, guildID)
			player, playerFound := bApp.StructsKeeper.GetPlayer(finalCtx, founder.Id)
			require.True(t, playerFound)
			if succeeds {
				require.Zero(t, result.Code, result.Log)
				require.True(t, found)
				require.Equal(t, founder.Id, guild.Owner)
				require.Equal(t, guildID, player.GuildId)
				require.Equal(t, count+1, bApp.StructsKeeper.GetGuildCount(finalCtx))
				require.Equal(t, initialVault-2000000, bApp.BankKeeper.GetBalance(finalCtx, vaults[0], "ualpha").Amount.Int64())
				require.Equal(t, int64(1000000), bApp.BankKeeper.GetBalance(finalCtx, treasury, denom).Amount.Int64())
    if scenario == "collateral_dust" {
     require.Equal(t, int64(1000001), bApp.BankKeeper.GetBalance(finalCtx, authtypes.NewModuleAddress(types.GuildBankCollateralPool + guildID), "ualpha").Amount.Int64())
    }
				require.Equal(t, int64(1000000), supply.Sub(bApp.BankKeeper.GetSupply(finalCtx, "ualpha").Amount).Int64())
				vaultEvents := 0
				for _, event := range result.Events {
					if event.Type == "structs.structs.EventVaultAddress" {
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						detail := decoded.(*types.EventVaultAddress).EventVaultAddressDetail
						require.Equal(t, guildID, detail.OwnerId)
						require.Equal(t, treasury.String(), detail.VaultAddress)
						vaultEvents++
					}
				}
				require.Equal(t, 1, vaultEvents)
			} else {
				require.NotZero(t, result.Code, result.Log)
				if scenario == "credit_refused" {
					require.Equal(t, 1, refusals)
					require.Contains(t, result.Log, "injected signed guild credit refusal")
				} else if scenario == "second_message_fails" {
					require.Contains(t, result.Log, "insufficient funds")
				} else if scenario == "jailed_reactor" {
 require.Contains(t, result.Log, "validator_jailed")
 } else if scenario == "underfunded" || scenario == "vesting_locked" {
					require.Contains(t, result.Log, "2000000ualpha")
					require.Zero(t, creditsAttempted)
					require.Equal(t, initialKey-1, bApp.BankKeeper.GetBalance(finalCtx, address, "ualpha").Amount.Int64())
				} else {
					require.Contains(t, result.Log, "permission")
				}
				require.False(t, found)
				require.Equal(t, count, bApp.StructsKeeper.GetGuildCount(finalCtx))
				require.Equal(t, founder.GuildId, player.GuildId)
				require.Equal(t, permissions, bApp.StructsKeeper.GetAllPermissionExport(finalCtx))
				require.Equal(t, initialVault, bApp.BankKeeper.GetBalance(finalCtx, vaults[0], "ualpha").Amount.Int64())
				require.Equal(t, supply, bApp.BankKeeper.GetSupply(finalCtx, "ualpha").Amount)
				require.True(t, bApp.BankKeeper.GetSupply(finalCtx, denom).IsZero())
				require.Nil(t, bApp.AccountKeeper.GetAccount(finalCtx, treasury))
				_, associated := bApp.StructsKeeper.GetVaultOwner(finalCtx, treasury)
				require.False(t, associated)
				for _, event := range result.Events {
					require.NotContains(t, []string{"burn", "coinbase", "structs.structs.EventGuild", "structs.structs.EventGuildBankAddress", "structs.structs.EventVaultAddress"}, event.Type)
				}
			}
			if vestingAccount != nil {
				require.Equal(t, vestingAccount, bApp.AccountKeeper.GetAccount(finalCtx, vaults[0]))
				require.True(t, bApp.BankKeeper.SpendableCoin(finalCtx, vaults[0], "ualpha").IsZero(), "locked funds remain inaccessible after the attempt")
			}
			if succeeds || scenario == "credit_refused" || scenario == "second_message_fails" {
				require.Equal(t, 1, creditsAttempted, "guild creation reached the final credit before a later message failed")
				require.Equal(t, initialKey-1, bApp.BankKeeper.GetBalance(finalCtx, address, "ualpha").Amount.Int64(), "ante fee remains separate from rolled-back vault funds")
			}
		})
	}
}
