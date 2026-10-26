package app_test

import (
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	"structs/x/structs/types"
)

// Signed restart keeps the paid meter but only schedules bounded restoration.
// Large reactors converge through the shared worker without a synchronous scan.
func TestSignedReactorRestartQueuesBoundedRestoration(t *testing.T) {
	for _, gas := range []uint64{ante.DefaultFreeGasCap, 60_000_000} {
		t.Run(math.NewIntFromUint64(gas).String(), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			reactorID := jailGateReactorId(t, bApp, ctx, validator)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)

			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			addresses := make([]sdk.AccAddress, 1000)
			for i := range addresses {
				address := make([]byte, 20)
				address[0] = 0x78
				binary.BigEndian.PutUint64(address[12:], uint64(i+1))
				addresses[i] = sdk.AccAddress(address)
				delegate(t, bApp, ctx, staking, denom, addresses[i], validator, math.NewInt(1000))
			}
			bApp.StructsKeeper.ReactorGateEnergy(ctx, validator)
			key := secp256k1.GenPrivKey()
			caller := sdk.AccAddress(key.PubKey().Address())
			player := appendAppPlayer(t, bApp.StructsKeeper, ctx, caller)
			fundRedelegationAccount(t, bApp, ctx, caller, sdk.NewCoins(sdk.NewInt64Coin(denom, 10)))
			account := bApp.AccountKeeper.GetAccount(ctx, caller)
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			messages := []sdk.Msg{&types.MsgReactorRestart{Creator: caller.String(), ValidatorAddress: validator.String()}}
			failedBatch := gas == 60_000_000
			if failedBatch {
				messages = append(messages, &types.MsgVaultSend{Creator: caller.String(), OwnerId: player.Id,
					RecipientOwnerId: player.Id, Amount: sdk.NewCoins(sdk.NewInt64Coin(denom, 1))})
			}
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config,
				messages,
				sdk.NewCoins(sdk.NewInt64Coin(denom, 1)), gas, jailGateChainID,
				[]uint64{account.GetAccountNumber()}, []uint64{account.GetSequence()}, key)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			require.Equal(t, int64(gas), result.GasWanted, "paid restart retains its requested meter")
			wantPower := uint64(1000)
			if failedBatch {
				require.NotZero(t, result.Code)
				require.Contains(t, result.Log, "insufficient funds")
				wantPower = 0
			} else {
				require.Zero(t, result.Code, result.Log)
			}
			require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap), "restart should not walk 1000 positions in the transaction")
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			if failedBatch {
				require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(finalCtx), "a failed later message rolls back restart scheduling")
				for _, event := range result.Events {
					require.NotContains(t, []string{"structs.structs.EventInfusion", "structs.structs.EventGrid"}, event.Type)
				}
			} else {
				require.NotEmpty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(finalCtx))
			}
			countRestored := func() int {
				restored := 0
				for _, address := range addresses {
					row, found := bApp.StructsKeeper.GetInfusion(finalCtx, reactorID, address.String())
					require.True(t, found)
					if row.Power != 0 {
						require.Equal(t, wantPower, row.Power)
						restored++
					}
				}
				return restored
			}
			targets, restoredEvents := map[string]bool{}, map[string]bool{}
			for _, address := range addresses {
				targets[address.String()] = true
			}
			collectRestorations := func(events []abci.Event) {
				for _, event := range events {
					if event.Type != "structs.structs.EventInfusion" {
						continue
					}
					// BaseApp block events include a raw mode attribute that is not
					// part of the typed protobuf payload.
					typed := abci.Event{Type: event.Type}
					for _, attribute := range event.Attributes {
						if attribute.Key == "infusion" {
							typed.Attributes = append(typed.Attributes, attribute)
						}
					}
					require.Len(t, typed.Attributes, 1)
					decoded, decodeErr := sdk.ParseTypedEvent(typed)
					require.NoError(t, decodeErr)
					row := decoded.(*types.EventInfusion).Infusion
					if !targets[row.Address] || row.Power == 0 {
						continue
					}
					require.Equal(t, reactorID, row.DestinationId)
					require.Equal(t, uint64(1000), row.Power)
					require.Equal(t, uint64(1000), row.Fuel)

					require.NotContains(t, restoredEvents, row.Address, "one public restoration update per position")
					restoredEvents[row.Address] = true
				}
			}
			collectRestorations(result.Events)
			collectRestorations(block.Events)
			previous := countRestored()
			require.LessOrEqual(t, previous, types.ReactorSlashReconcileBudget)
			queueBeforeCommit := bApp.StructsKeeper.GetReactorSlashReconcileQueue(finalCtx)
			_, err = bApp.Commit()
			require.NoError(t, err)
			height, blockTime := int64(1), ctx.BlockTime()
			finalCtx = bApp.NewContextLegacy(true, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: blockTime})
			require.Equal(t, queueBeforeCommit, bApp.StructsKeeper.GetReactorSlashReconcileQueue(finalCtx))
			feesPaid := int64(1)
			for pass := 0; (pass == 0 || len(bApp.StructsKeeper.GetReactorSlashReconcileQueue(finalCtx)) > 0) && pass < 20; pass++ {
				height++
				blockTime = blockTime.Add(time.Second)
				var repeated [][]byte
				// Stop repeating once the remaining batch can complete, avoiding
				// a genuinely new repair request after that completion.
				if !failedBatch && len(addresses)-previous > types.ReactorSlashReconcileBudget {
					signer := bApp.AccountKeeper.GetAccount(finalCtx, caller)
					retry, signErr := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages,
						sdk.NewCoins(sdk.NewInt64Coin(denom, 1)), gas, jailGateChainID,
						[]uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
					require.NoError(t, signErr)
					encoded, encodeErr := config.TxEncoder()(retry)
					require.NoError(t, encodeErr)
					repeated = append(repeated, encoded)
				}
				nextBlock, finalizeErr := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: blockTime, Txs: repeated})
				require.NoError(t, finalizeErr)
				require.Len(t, nextBlock.TxResults, len(repeated))
				for _, retry := range nextBlock.TxResults {
					require.Zero(t, retry.Code, retry.Log)
					require.Less(t, retry.GasUsed, int64(ante.DefaultFreeGasCap))
					feesPaid++
					collectRestorations(retry.Events)
				}
				collectRestorations(nextBlock.Events)
				_, err = bApp.Commit()
				require.NoError(t, err)
				finalCtx = bApp.NewContextLegacy(true, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: blockTime})
				restored := countRestored()
				require.LessOrEqual(t, restored-previous, types.ReactorSlashReconcileBudget)
				if !failedBatch {
					require.Greater(t, restored, previous, "repeated signed restarts must not rewind completed work")
				}
				previous = restored
			}
			require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(finalCtx))
			if failedBatch {
				require.Zero(t, previous)
			} else {
				require.Equal(t, len(addresses), previous)
				require.Equal(t, int64(8), height, "repeated requests do not delay the existing recovery schedule")
				require.Greater(t, feesPaid, int64(1))
			}
			require.Len(t, restoredEvents, previous, "committed restoration events match the recovered positions")
			cc := bApp.StructsKeeper.NewCurrentContext(finalCtx)
			for _, address := range addresses {
				row, found := bApp.StructsKeeper.GetInfusion(finalCtx, reactorID, address.String())
				require.True(t, found)
				require.Equal(t, uint64(1000), row.Fuel)

				require.Equal(t, wantPower, row.Power, address.String())
				player, err := cc.GetPlayerByAddress(address.String())
				require.NoError(t, err)
				_, _, playerPower := row.GetPowerDistribution()
				require.Equal(t, playerPower, player.GetCapacity(), "capacity must follow the committed infusion, including on rollback")
			}
			require.Equal(t, int64(10)-feesPaid, bApp.BankKeeper.GetBalance(finalCtx, caller, denom).Amount.Int64(), "the scheduling transaction retains its ante fee")
			requireStakingAndDistributionHealthy(t, bApp, finalCtx)
			t.Logf("positions=%d gas_wanted=%d gas_used=%d code=%d final_height=%d fees_paid=%d", len(addresses), result.GasWanted, result.GasUsed, result.Code, height, feesPaid)
		})
	}
}

// A caller can requeue a small, already-repaired reactor every block. Its work
// must share the budget without preventing a larger reactor from converging.
func TestSignedRepeatedSmallReactorRestartDoesNotStarveLargeReactor(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	require.Len(t, validators, 2)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	validatorAddresses := make([]sdk.ValAddress, 2)
	reactorIDs := make([]string, 2)
	positions := make([][]sdk.AccAddress, 2)
	expectedPower := make([]uint64, 2)
	for i := range validators {
		validatorAddresses[i], err = sdk.ValAddressFromBech32(validators[i].OperatorAddress)
		require.NoError(t, err)
		reactorIDs[i] = jailGateReactorId(t, bApp, ctx, validatorAddresses[i])
	}
	// Keep the repeatedly submitted small reactor first in store order.
	if reactorIDs[0] > reactorIDs[1] {
		reactorIDs[0], reactorIDs[1] = reactorIDs[1], reactorIDs[0]
		validatorAddresses[0], validatorAddresses[1] = validatorAddresses[1], validatorAddresses[0]
	}
	for i, count := range []int{16, 4 * types.ReactorSlashReconcileBudget} {
		positions[i] = make([]sdk.AccAddress, count)
		for j := range positions[i] {
			address := make([]byte, 20)
			address[0], address[1] = 0x79, byte(i)
			binary.BigEndian.PutUint64(address[12:], uint64(j+1))
			positions[i][j] = sdk.AccAddress(address)
			delegate(t, bApp, ctx, staking, denom, positions[i][j], validatorAddresses[i], math.NewInt(1000))
		}
		before, found := bApp.StructsKeeper.GetInfusion(ctx, reactorIDs[i], positions[i][0].String())
		require.True(t, found)
		require.Positive(t, before.Power)
		expectedPower[i] = before.Power
		bApp.StructsKeeper.ReactorGateEnergy(ctx, validatorAddresses[i])
		for _, address := range positions[i] {
			gated, found := bApp.StructsKeeper.GetInfusion(ctx, reactorIDs[i], address.String())
			require.True(t, found)
			require.Zero(t, gated.Power)
		}
	}
	key := secp256k1.GenPrivKey()
	caller := sdk.AccAddress(key.PubKey().Address())
	appendAppPlayer(t, bApp.StructsKeeper, ctx, caller)
	fundRedelegationAccount(t, bApp, ctx, caller, sdk.NewCoins(sdk.NewInt64Coin(denom, 20)))
	config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
	current := ctx
	previous := [2]int{}
	fees := int64(0)
	for height := int64(1); height <= 10; height++ {
		messages := []sdk.Msg{&types.MsgReactorRestart{Creator: caller.String(), ValidatorAddress: validatorAddresses[0].String()}}
		if height == 1 {
			messages = append(messages, &types.MsgReactorRestart{Creator: caller.String(), ValidatorAddress: validatorAddresses[1].String()})
		}
		account := bApp.AccountKeeper.GetAccount(current, caller)
		tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(height)), config, messages,
			sdk.NewCoins(sdk.NewInt64Coin(denom, 1)), ante.DefaultFreeGasCap, jailGateChainID,
			[]uint64{account.GetAccountNumber()}, []uint64{account.GetSequence()}, key)
		require.NoError(t, err)
		wire, err := config.TxEncoder()(tx)
		require.NoError(t, err)
		blockTime := ctx.BlockTime().Add(time.Duration(height) * time.Second)
		block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: blockTime, Txs: [][]byte{wire}})
		require.NoError(t, err)
		require.Len(t, block.TxResults, 1)
		require.Zero(t, block.TxResults[0].Code, block.TxResults[0].Log)
		require.Less(t, block.TxResults[0].GasUsed, int64(ante.DefaultFreeGasCap))
		fees++
		current = bApp.NewContext(false).WithBlockTime(blockTime)
		restored := [2]int{}
		for i := range positions {
			for _, address := range positions[i] {
				row, found := bApp.StructsKeeper.GetInfusion(current, reactorIDs[i], address.String())
				require.True(t, found)
				require.Equal(t, uint64(1000), row.Fuel)

				require.Contains(t, []uint64{0, expectedPower[i]}, row.Power)
				if row.Power != 0 {
					restored[i]++
				}
			}
		}
		require.Greater(t, restored[1], previous[1], "large reactor must progress despite repeated small-reactor requests")
		require.Equal(t, len(positions[0]), restored[0], "small reactor finishes in the first shared batch")
		require.LessOrEqual(t, restored[0]+restored[1]-previous[0]-previous[1], types.ReactorSlashReconcileBudget)
		previous = restored
		_, err = bApp.Commit()
		require.NoError(t, err)
		current = bApp.NewContextLegacy(true, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: blockTime})
		if restored[1] == len(positions[1]) && len(bApp.StructsKeeper.GetReactorSlashReconcileQueue(current)) == 0 {
			break
		}
	}
	require.Equal(t, len(positions[1]), previous[1])
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(current))
	require.Greater(t, fees, int64(1))
	require.Equal(t, int64(20)-fees, bApp.BankKeeper.GetBalance(current, caller, denom).Amount.Int64())
	requireStakingAndDistributionHealthy(t, bApp, current)
	t.Logf("small_positions=%d large_positions=%d committed_blocks=%d fees_paid=%d", len(positions[0]), len(positions[1]), fees, fees)
}
