package app_test

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Measures handler/store gas in discarded caches, then delivers a signed
// transaction including ante overhead. These simple fixtures must fit the cap;
// they do not establish a bound for other counts, denominations or schedules.
func TestVaultVestingInfusionGasByScheduleCount(t *testing.T) {
	for _, count := range []int{100, 1000, 10000} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			primary := sdk.AccAddress(key.PubKey().Address())
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			playerID := cc.UpsertPlayer(primary.String()).GetPlayerId()
			cc.CommitAll()
			addresses, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID)
			require.NoError(t, err)
			vault := addresses[0]
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			account := &types.VaultVestingAccount{
				BaseAccount: bApp.AccountKeeper.GetAccount(ctx, vault).(*authtypes.BaseAccount),
				OwnerId:     playerID,
			}
			for i := 0; i < count; i++ {
				address := make([]byte, 20)
				binary.BigEndian.PutUint64(address[12:], uint64(i+1))
				account.Schedules = append(account.Schedules, &types.VaultVestingSchedule{
					SourceAddress: address, Kind: 2,
					EndTime:         ctx.BlockTime().Add(365 * 24 * time.Hour).Unix(),
					OriginalVesting: sdk.NewCoins(sdk.NewInt64Coin(denom, 1000)),
				})
			}
			require.NoError(t, account.Validate())
			bApp.AccountKeeper.SetAccount(ctx, account)
			fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin(denom, int64(count)*1000)))
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			message := &types.MsgReactorInfuse{Creator: primary.String(), ValidatorAddress: validators[0].OperatorAddress, Amount: sdk.NewInt64Coin(denom, 1000)}
			server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			measured, _ := ctx.CacheContext()
			meter := storetypes.NewInfiniteGasMeter()
			measured = measured.WithGasMeter(meter)
			_, err = server.ReactorInfuse(measured, message)
			require.NoError(t, err)
			gas := meter.GasConsumed()
			wire, err := account.Marshal()
			require.NoError(t, err)
			t.Logf("schedules=%d account_bytes=%d handler_gas=%d free_cap=%d fits=%t", count, len(wire), gas, ante.DefaultFreeGasCap, gas <= ante.DefaultFreeGasCap)
			require.LessOrEqual(t, gas, ante.DefaultFreeGasCap, "simple delayed-schedule handler gas regression")
			limited, _ := ctx.CacheContext()
			limited = limited.WithGasMeter(storetypes.NewGasMeter(ante.DefaultFreeGasCap))
			var caught any
			func() {
				defer func() { caught = recover() }()
				_, err = server.ReactorInfuse(limited, message)
			}()
			if gas > ante.DefaultFreeGasCap {
				require.IsType(t, storetypes.ErrorOutOfGas{}, caught)
			} else {
				require.Nil(t, caught)
				require.NoError(t, err)
			}
			require.Equal(t, account, bApp.AccountKeeper.GetAccount(ctx, vault), "discarded execution cannot change parent vesting counters")
			// Deliver a signed transaction through the real ante chain and gas
			// router as well; handler-only measurements omit those costs.
			signer := bApp.AccountKeeper.GetAccount(ctx, primary)
			txConfig := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), txConfig,
				[]sdk.Msg{message}, nil, ante.DefaultFreeGasCap, jailGateChainID,
				[]uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, err)
			txBytes, err := txConfig.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{txBytes}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			delivered := block.TxResults[0]
			require.Zero(t, delivered.Code, "signed infusion failed: %s", delivered.Log)
			t.Logf("schedules=%d signed_tx_gas=%d", count, delivered.GasUsed)
			require.LessOrEqual(t, delivered.GasUsed, int64(ante.DefaultFreeGasCap))
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			installed := bApp.AccountKeeper.GetAccount(finalCtx, vault).(*types.VaultVestingAccount)
			require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 1000)), installed.GetDelegatedVesting())
			require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, int64(count)*1000-1000)), installed.LockedCoins(ctx.BlockTime()))
		})
	}
}
