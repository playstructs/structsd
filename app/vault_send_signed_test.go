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
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSignedVaultGuildTokenEscrowRefusalRollsBackEarlierSend(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		name := "player"
		if guildOwned { name = "guild" }
		t.Run(name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			peer := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x75}, 20)))
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-915", 915, actor.Id, actor.Creator
			bApp.StructsKeeper.SetGuild(ctx, guild)
			ownerID := actor.Id
			if guildOwned { ownerID = guild.Id }
			source, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(ownerID)
			require.NoError(t, err)
			denom := "uguild." + guild.Id
			fundRedelegationAccount(t, bApp, ctx, source[0], sdk.NewCoins(sdk.NewInt64Coin(denom, 100)))
			destination, err := types.VaultAddress(peer.Id)
			require.NoError(t, err)
			require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, destination))
			escrow := ibctransfertypes.GetEscrowAddress(ibctransfertypes.PortID, "channel-0")
			messages := []sdk.Msg{
				&types.MsgVaultSend{Creator: address.String(), OwnerId: ownerID, RecipientOwnerId: peer.Id,
					Amount: sdk.NewCoins(sdk.NewInt64Coin(denom, 10))},
				&types.MsgPlayerSend{Creator: address.String(), FromAddress: source[0].String(), ToAddress: escrow.String(),
					Amount: sdk.NewCoins(sdk.NewInt64Coin(denom, 10))},
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
			require.NotZero(t, result.Code)
			require.Contains(t, result.Log, "message index: 1", "the first transfer must reach execution before the refusal")
			require.Contains(t, result.Log, "recipient_not_eligible")
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(finalCtx, source[0], denom).Amount.Int64())
			require.True(t, bApp.BankKeeper.GetBalance(finalCtx, destination, denom).IsZero())
			require.True(t, bApp.BankKeeper.GetBalance(finalCtx, escrow, denom).IsZero())
			require.Equal(t, int64(100), bApp.BankKeeper.GetSupply(finalCtx, denom).Amount.Int64())
			require.Nil(t, bApp.AccountKeeper.GetAccount(finalCtx, destination))
			_, found := bApp.StructsKeeper.GetVaultOwner(finalCtx, destination)
			require.False(t, found, "failed batch cannot leave a newly provisioned vault association")
			owner, found := bApp.StructsKeeper.GetVaultOwner(finalCtx, source[0])
			require.True(t, found)
			require.Equal(t, ownerID, owner)
			for _, event := range result.Events {
				require.NotContains(t, []string{"transfer", "coin_spent", "coin_received", "structs.structs.EventVaultAddress"}, event.Type)
			}
		})
	}
}

func TestSignedVaultSendAuthorityAndBatchRollback(t *testing.T) {
	testSignedVaultSend(t, false)
}

func TestSignedPlayerSendFromVaultFundsEmptySigningKey(t *testing.T) {
	testSignedVaultSend(t, true)
}

func testSignedVaultSend(t *testing.T, external bool) {
	t.Helper()
	for _, kind := range []string{"player", "guild"} {
		for _, scenario := range []string{"success", "restricted_key", "outsider", "insufficient_second_send"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				bApp, ctx := setupJailGateApp(t)
				key := secp256k1.GenPrivKey()
				address := sdk.AccAddress(key.PubKey().Address())
				actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
				peer := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x75}, 20)))
				victim := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x76}, 20)))
				ownerID := actor.Id
				if scenario == "outsider" {
					ownerID = victim.Id
				}
				if kind == "guild" {
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner, guild.Creator = "0-915", 915, ownerID, actor.Creator
					bApp.StructsKeeper.SetGuild(ctx, guild)
					ownerID = guild.Id
				}
				source, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(ownerID)
				require.NoError(t, err)
				fundRedelegationAccount(t, bApp, ctx, source[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
				destination, err := types.VaultAddress(peer.Id)
				require.NoError(t, err)
				if external {
					destination = address
					require.True(t, bApp.BankKeeper.GetAllBalances(ctx, address).IsZero(), "fee funding must work from an empty signing key")
				}
				if scenario == "restricted_key" {
					bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermTokenTransfer)
				}
				var message sdk.Msg = &types.MsgVaultSend{Creator: address.String(), OwnerId: ownerID, RecipientOwnerId: peer.Id,
					Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 80))}
				if external {
					message = &types.MsgPlayerSend{Creator: address.String(), FromAddress: source[0].String(), ToAddress: destination.String(), Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 80))}
				}
				messages := []sdk.Msg{message}
				if scenario == "insufficient_second_send" {
					messages = append(messages, message)
				}
				signer := bApp.AccountKeeper.GetAccount(ctx, address)
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
					ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
				require.NoError(t, err)
				encoded, err := config.TxEncoder()(tx)
				require.NoError(t, err)
				txs := [][]byte{encoded}
				payNativeFee := external && scenario == "success"
				if payNativeFee {
					native := &banktypes.MsgSend{FromAddress: address.String(), ToAddress: peer.PrimaryAddress, Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1))}
					paid, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(2)), config, []sdk.Msg{native}, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1)),
						ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()+1}, key)
					require.NoError(t, err)
					wire, err := config.TxEncoder()(paid)
					require.NoError(t, err)
					txs = append(txs, wire)
				}
				block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: txs})
				require.NoError(t, err)
				require.Len(t, block.TxResults, len(txs))
				result := block.TxResults[0]
				finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
				if scenario == "success" {
					require.Zero(t, result.Code, result.Log)
					require.Equal(t, int64(20), bApp.BankKeeper.GetBalance(finalCtx, source[0], "ualpha").Amount.Int64())
					expectedBalance := int64(80)
					if payNativeFee {
						require.Zero(t, block.TxResults[1].Code, block.TxResults[1].Log)
						expectedBalance = 78 // One unit sent and one unit paid as SDK fees.
					}
					require.Equal(t, expectedBalance, bApp.BankKeeper.GetBalance(finalCtx, destination, "ualpha").Amount.Int64())
					if external {
						require.Equal(t, actor.Index, bApp.StructsKeeper.GetPlayerIndexFromAddress(finalCtx, address.String()))
					} else {
						owner, found := bApp.StructsKeeper.GetVaultOwner(finalCtx, destination)
						require.True(t, found)
						require.Equal(t, peer.Id, owner)
						require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(finalCtx, destination.String()))
					}
				} else {
					require.NotZero(t, result.Code)
					if scenario == "insufficient_second_send" {
						require.Contains(t, result.Log, "message index: 1")
						require.Contains(t, result.Log, "insufficient")
					} else {
						require.Contains(t, result.Log, "permission")
					}
					require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(finalCtx, source[0], "ualpha").Amount.Int64())
					require.True(t, bApp.BankKeeper.GetBalance(finalCtx, destination, "ualpha").IsZero())
					if !external {
						require.Nil(t, bApp.AccountKeeper.GetAccount(finalCtx, destination))
					}
					_, found := bApp.StructsKeeper.GetVaultOwner(finalCtx, destination)
					require.False(t, found)
					for _, event := range result.Events {
						require.NotEqual(t, "structs.structs.EventVaultAddress", event.Type)
						require.NotEqual(t, "transfer", event.Type)
					}
				}
			})
		}
	}
}
