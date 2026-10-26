package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestRetainedNonOperatorVestingSurvivesExportAndInitChain(t *testing.T) {
	for _, zeroHeight := range []bool{false, true} {
		name := "preserve-height"
		if zeroHeight {
			name = "zero-height"
		}
		t.Run(name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			primary, _, playerID := newTransferPlayer(t, bApp, ctx)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			coins := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin(denom, n)) }
			base := bApp.AccountKeeper.GetAccount(ctx, primary).(*authtypes.BaseAccount)
			end := ctx.BlockTime().Add(365 * 24 * time.Hour)
			source, err := vestingtypes.NewDelayedVestingAccount(base, coins(1000000), end.Unix())
			require.NoError(t, err)
			bApp.AccountKeeper.SetAccount(ctx, source)
			fundRedelegationAccount(t, bApp, ctx, primary, coins(400000))
			server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(600000))
			_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), coins(200000)[0]))
			require.NoError(t, err)
			require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
			vault, err := types.VaultAddress(playerID)
			require.NoError(t, err)
			before := bApp.AccountKeeper.GetAccount(ctx, primary).(*vestingtypes.DelayedVestingAccount)
			beforeWire, err := before.Marshal()
			require.NoError(t, err)
			_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
			require.NoError(t, err)
			_, err = bApp.Commit()
			require.NoError(t, err)
			exported, err := bApp.ExportAppStateAndValidators(zeroHeight, nil, nil)
			require.NoError(t, err)
			restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
			require.NoError(t, err)
			restartTime := ctx.BlockTime().Add(time.Second)
			_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: restartTime, ConsensusParams: &exported.ConsensusParams})
			require.NoError(t, err)
			height := exported.Height
			if height == 0 {
				height = 1
			}
			restored := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: restartTime})
			after := restarted.AccountKeeper.GetAccount(restored, primary).(*vestingtypes.DelayedVestingAccount)
			afterWire, err := after.Marshal()
			require.NoError(t, err)
			require.Equal(t, beforeWire, afterWire)
			require.NoError(t, after.Validate())
			require.Equal(t, before, restarted.AccountKeeper.GetAccount(restored, primary))
			require.Equal(t, coins(400000), after.LockedCoins(restartTime))
			owner, found := restarted.StructsKeeper.GetVaultOwner(restored, vault)
			require.True(t, found)
			require.Equal(t, playerID, owner)
			require.Zero(t, restarted.StructsKeeper.GetPlayerIndexFromAddress(restored, vault.String()))
			_, err = restarted.DistrKeeper.WithdrawDelegationRewards(restored, primary, validator)
			require.NoError(t, err)
			ms := keeperlib.NewMsgServerImpl(restarted.StructsKeeper)
			_, err = ms.ReactorInfuse(restored, &types.MsgReactorInfuse{Creator: primary.String(), DelegatorAddress: primary.String(), ValidatorAddress: validator.String(), Amount: coins(100000)[0]})
			require.NoError(t, err)
			ubd, err := restarted.StakingKeeper.GetUnbondingDelegation(restored, primary, validator)
			require.NoError(t, err)
			require.Len(t, ubd.Entries, 1)
			mature := restored.WithBlockTime(ubd.Entries[0].CompletionTime.Add(time.Second))
			paid, err := restarted.StakingKeeper.CompleteUnbonding(mature, primary, validator)
			require.NoError(t, err)
			require.Equal(t, coins(200000), paid)
			continued := restarted.AccountKeeper.GetAccount(mature, primary).(*vestingtypes.DelayedVestingAccount)
			require.Equal(t, coins(500000), continued.GetDelegatedVesting())
			require.Equal(t, coins(500000), continued.LockedCoins(mature.BlockTime()))
			require.True(t, continued.LockedCoins(end).IsZero())
			requireStakingAndDistributionHealthy(t, restarted, mature)
		})
	}
}
