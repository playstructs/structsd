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
	"structs/x/structs/types"
)

func TestSignedVaultRotationAndFailedBatchRollback(t *testing.T) {
	for _, failSecond := range []bool{false, true} {
		name := "success"
		if failSecond {
			name = "failed_second_message"
		}
		t.Run(name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			primary := sdk.AccAddress(key.PubKey().Address())
			player := appendAppPlayer(t, bApp.StructsKeeper, ctx, primary)
			secondary := sdk.AccAddress(bytes.Repeat([]byte{0x43}, 20))
			require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, secondary.String(), player.Index))
			outsider := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x44}, 20)))
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(player.Id)
			require.NoError(t, err)
			vault := vaults[0]
			delegate(t, bApp, ctx, staking, denom, primary, validator, math.NewInt(1000000))
			delegate(t, bApp, ctx, staking, denom, vault, validator, math.NewInt(1000000))
			fundRedelegationAccount(t, bApp, ctx, primary, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 123)))
			reactorID := jailGateReactorId(t, bApp, ctx, validator)

			sourceBefore, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
			require.True(t, found)
			vaultBefore, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
			require.True(t, found)
			sourceStake, err := bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
			require.NoError(t, err)
			vaultStake, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
			require.NoError(t, err)
			capacity := redelegationCapacity(t, bApp, ctx, player.Id)
			messages := []sdk.Msg{&types.MsgPlayerUpdatePrimaryAddress{Creator: primary.String(), PrimaryAddress: secondary.String()}}
			if failSecond {
				messages = append(messages, &types.MsgPlayerUpdatePrimaryAddress{Creator: primary.String(), PrimaryAddress: outsider.PrimaryAddress})
			}
			signer := bApp.AccountKeeper.GetAccount(ctx, primary)
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil, ante.DefaultFreeGasCap, jailGateChainID,
				[]uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			restored := bApp.StructsKeeper.NewCurrentContext(finalCtx).GetPlayer(player.Id)
			merged, found := bApp.StructsKeeper.GetInfusion(finalCtx, reactorID, vault.String())
			require.True(t, found)
			finalVaultStake, err := bApp.StakingKeeper.GetDelegation(finalCtx, vault, validator)
			require.NoError(t, err)
			if failSecond {
				finalSourceStake, err := bApp.StakingKeeper.GetDelegation(finalCtx, primary, validator)
				require.NoError(t, err)
				require.Equal(t, sourceStake, finalSourceStake)
				require.Equal(t, vaultStake, finalVaultStake)
				require.NotZero(t, result.Code)
				require.Contains(t, result.Log, "message index: 1")
				require.Equal(t, primary.String(), restored.GetPrimaryAddress())
				require.Equal(t, vaultBefore, merged)
				source, found := bApp.StructsKeeper.GetInfusion(finalCtx, reactorID, primary.String())
				require.True(t, found)
				require.Equal(t, sourceBefore, source)
				require.Equal(t, int64(123), bApp.BankKeeper.GetBalance(finalCtx, primary, "ualpha").Amount.Int64())
				require.True(t, bApp.BankKeeper.GetBalance(finalCtx, vault, "ualpha").IsZero())
				for _, event := range result.Events {
					require.NotEqual(t, "transfer", event.Type)
					require.NotEqual(t, "structs.structs.EventInfusion", event.Type)
				}
			} else {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, sourceStake.Shares.Add(vaultStake.Shares), finalVaultStake.Shares)
				_, err = bApp.StakingKeeper.GetDelegation(finalCtx, primary, validator)
				require.Error(t, err)
				require.Equal(t, secondary.String(), restored.GetPrimaryAddress())

				require.Equal(t, uint64(2000000), merged.Fuel)
				require.Equal(t, int64(123), bApp.BankKeeper.GetBalance(finalCtx, vault, "ualpha").Amount.Int64())
				require.True(t, bApp.BankKeeper.GetBalance(finalCtx, primary, "ualpha").IsZero())
			}
			require.Equal(t, capacity, redelegationCapacity(t, bApp, finalCtx, player.Id))
			requireStakingAndDistributionHealthy(t, bApp, finalCtx)
		})
	}
}
