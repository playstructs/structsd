package app_test

import (
	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/x/structs/types"
	"testing"
	"time"
)

func TestReactorStakeCapActivationIsStickyAcrossCountLossAndRestart(t *testing.T) {
	for _, zero := range []bool{false, true} {
		t.Run(map[bool]string{true: "zero-height", false: "preserve-height"}[zero], func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 4)
			require.False(t, bApp.StructsKeeper.ReactorStakeCapActive(ctx), "production threshold is ten")
			p := bApp.StructsKeeper.GetParams(ctx)
			p.ReactorStakeCapMinValidators = 4
			require.NoError(t, bApp.StructsKeeper.SetParams(ctx, p))
			// The hook count is evaluated by Structs after the installed staking set.
			_, err := bApp.StructsKeeper.EndBlocker(ctx)
			require.NoError(t, err)
			require.True(t, bApp.StructsKeeper.ReactorStakeCapActive(ctx))
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			for _, validator := range validators[:3] {
				cons, err := validator.GetConsAddr()
				require.NoError(t, err)
				require.NoError(t, bApp.StakingKeeper.Jail(ctx, cons))
			}
			_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
			require.NoError(t, err)
			validators, err = bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			require.Len(t, validators, 1)
			require.NoError(t, bApp.StructsKeeper.EvaluateReactorStakeCap(ctx))
			require.True(t, bApp.StructsKeeper.ReactorStakeCapActive(ctx))
			_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
			require.NoError(t, err)
			_, err = bApp.Commit()
			require.NoError(t, err)
			exported, err := bApp.ExportAppStateAndValidators(zero, nil, nil)
			require.NoError(t, err)
			restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
			require.NoError(t, err)
			now := ctx.BlockTime().Add(time.Second)
			_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: now, ConsensusParams: &exported.ConsensusParams})
			require.NoError(t, err)
			height := exported.Height
			if height == 0 {
				height = 1
			}
			restored := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: now})
			require.True(t, restarted.StructsKeeper.ReactorStakeCapActive(restored))
			require.Equal(t, p, restarted.StructsKeeper.GetParams(restored))
			address, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			query, err := restarted.StructsKeeper.ReactorStakeShare(restored, &types.QueryReactorStakeShareRequest{ReactorId: jailGateReactorId(t, restarted, restored, address)})
			require.NoError(t, err)
			require.True(t, query.CapActive)
			require.Equal(t, "0", query.ExternalDepositHeadroom)
			require.Equal(t, "0", query.BondedRedelegationHeadroom)
			requireStakingAndDistributionHealthy(t, restarted, restored)
		})
	}
}
