package app_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/x/structs/types"
)

func TestStakingOperationSequenceAndValidatorHoldSurviveGenesis(t *testing.T) {
	for _, zeroHeight := range []bool{false, true} {
		for _, pending := range []bool{false, true} {
			t.Run(fmt.Sprintf("zero=%t/pending=%t", zeroHeight, pending), func(t *testing.T) {
				bApp, ctx := setupJailGateApp(t)
				var validatorID uint64
				if pending {
					address := newBondedValidator(t, bApp, ctx, "held-validator")
					validator, err := bApp.StakingKeeper.GetValidator(ctx, address)
					require.NoError(t, err)
					consAddr, err := validator.GetConsAddr()
					require.NoError(t, err)
					require.NoError(t, bApp.StakingKeeper.Jail(ctx, consAddr))
					_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
					require.NoError(t, err)
					validator, err = bApp.StakingKeeper.GetValidator(ctx, address)
					require.NoError(t, err)
					require.True(t, validator.IsUnbonding())
					require.Len(t, validator.UnbondingIds, 1)
					validatorID = validator.UnbondingIds[0]
					require.NoError(t, bApp.StakingKeeper.PutUnbondingOnHold(ctx, validatorID))
				}
				// Consumed IDs with no live rows model completed operations. An
				// import that only takes max(live IDs) would reuse this sequence.
				var last uint64
				for i := 0; i < 7; i++ {
					var err error
					last, err = bApp.StakingKeeper.IncrementUnbondingID(ctx)
					require.NoError(t, err)
				}
				_, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
				require.NoError(t, err)
				_, err = bApp.Commit()
				require.NoError(t, err)
				exported, err := bApp.ExportAppStateAndValidators(zeroHeight, nil, nil)
				require.NoError(t, err)
				var modules map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(exported.AppState, &modules))
				var genesis types.GenesisState
				require.NoError(t, bApp.AppCodec().UnmarshalJSON(modules[types.ModuleName], &genesis))
				require.Equal(t, last, genesis.StakingUnbondingId)
				restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
				require.NoError(t, err)
				when := ctx.BlockTime().Add(time.Second)
				_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: when, ConsensusParams: &exported.ConsensusParams})
				require.NoError(t, err)
				height := exported.Height
				if height == 0 {
					height = 1
				}
				restored := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: when})
				next, err := restarted.StakingKeeper.IncrementUnbondingID(restored)
				require.NoError(t, err)
				require.Equal(t, last+1, next)
				if pending {
					kind, err := restarted.StakingKeeper.GetUnbondingType(restored, validatorID)
					require.NoError(t, err)
					require.Equal(t, stakingtypes.UnbondingType_ValidatorUnbonding, kind)
					validator, err := restarted.StakingKeeper.GetValidatorByUnbondingID(restored, validatorID)
					require.NoError(t, err)
					require.EqualValues(t, 1, validator.UnbondingOnHoldRefCount)
					require.NoError(t, restarted.StakingKeeper.UnbondingCanComplete(restored, validatorID))
					validator, err = restarted.StakingKeeper.GetValidatorByUnbondingID(restored, validatorID)
					require.NoError(t, err)
					require.Zero(t, validator.UnbondingOnHoldRefCount)
				}
				_, err = restarted.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: when})
				require.NoError(t, err)
				_, err = restarted.Commit()
				require.NoError(t, err)
			})
		}
	}
}
