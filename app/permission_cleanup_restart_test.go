package app_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestPermissionCleanupContinuesAcrossAppRestart(t *testing.T) {
	for _, zeroHeight := range []bool{false, true} {
		t.Run(fmt.Sprintf("zero_height_%t", zeroHeight), func(t *testing.T) {
			original, ctx := setupJailGateApp(t)
			const objectID = "6-999"
			original.StructsKeeper.SetAllocationCount(ctx, 1000)
			const grants = types.PermissionCleanupBudget*2 + 10
			for i := 0; i < grants; i++ {
				original.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(objectID, fmt.Sprintf("1-%d", i+1)), types.PermPlay)
				original.StructsKeeper.SetGuildRankPermissionStoreOnly(ctx, objectID, fmt.Sprintf("0-%d", i+1), types.PermPlay, 3)
			}
			// The caller's grant sorts after the filler rows, so it remains
			// present both at export and during the first restarted transaction.
			original.StructsKeeper.SetPlayerCount(ctx, 999999)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			actor := appendAppPlayer(t, original.StructsKeeper, ctx, address)
			grantID := keeperlib.GetObjectPermissionIDBytes(objectID, actor.Id)
			original.StructsKeeper.SetPermissionsByBytes(ctx, grantID, types.PermAdmin)
			original.StructsKeeper.EnqueuePermissionCleanup(ctx, objectID)
			_, err := original.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
			require.NoError(t, err)
			_, err = original.Commit()
			require.NoError(t, err)
			exported, err := original.ExportAppStateAndValidators(zeroHeight, nil, nil)
			require.NoError(t, err)
			var modules map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(exported.AppState, &modules))
			var state types.GenesisState
			original.AppCodec().MustUnmarshalJSON(modules[types.ModuleName], &state)
			require.Equal(t, []string{objectID}, state.PermissionCleanupQueue)
			remainingDirect := 0
			expectedDirect := make(map[string]bool)
			expectedRanks := make(map[string]bool)
			for i := 0; i < grants; i++ {
				expectedRanks[fmt.Sprintf("0-%d", i+1)] = true
			}
			for _, permission := range state.PermissionList {
				if strings.HasPrefix(permission.PermissionId, objectID+"@") {
					remainingDirect++
					expectedDirect[permission.PermissionId] = true
				}
			}
			require.Equal(t, grants+1-(types.PermissionCleanupBudget-1), remainingDirect)

			options := simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}
			restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, options, baseapp.SetChainID(jailGateChainID))
			require.NoError(t, err)
			restartTime := ctx.BlockTime().Add(time.Second)
			_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: restartTime, ConsensusParams: &exported.ConsensusParams})
			require.NoError(t, err)
			height := exported.Height
			if height == 0 {
				height = 1
			}
			restoredCtx := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: restartTime})
			require.Equal(t, types.PermAdmin, restarted.StructsKeeper.GetPermissionsByBytes(restoredCtx, grantID))
			account := restarted.AccountKeeper.GetAccount(restoredCtx, address)
			require.NotNil(t, account)
			config := authtx.NewTxConfig(restarted.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config,
				[]sdk.Msg{&types.MsgAllocationTransfer{Creator: address.String(), AllocationId: objectID, Controller: actor.Id}},
				sdk.NewCoins(), 1000000, jailGateChainID,
				[]uint64{account.GetAccountNumber()}, []uint64{account.GetSequence()}, key)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			directEvents, rankEvents := 0, 0
			for block := 0; block < 8; block++ {
				now := restartTime.Add(time.Duration(block) * time.Second)
				request := &abci.RequestFinalizeBlock{Height: height + int64(block), Time: now}
				if block == 0 {
					request.Txs = [][]byte{wire}
				}
				result, err := restarted.FinalizeBlock(request)
				require.NoError(t, err)
				if block == 0 {
					require.Len(t, result.TxResults, 1)
					require.NotZero(t, result.TxResults[0].Code)
					require.Contains(t, result.TxResults[0].Log, "allocation (6-999) not found")
					afterTx := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: now})
					require.Equal(t, types.PermAdmin, restarted.StructsKeeper.GetPermissionsByBytes(afterTx, grantID), "refusal must be observed while the old grant still exists")
					_, exists := restarted.StructsKeeper.GetAllocation(afterTx, objectID)
					require.False(t, exists)
					for _, event := range result.TxResults[0].Events {
						require.NotEqual(t, "structs.structs.EventAllocation", event.Type)
						require.NotEqual(t, "structs.structs.EventPermission", event.Type)
					}
				}
				for _, event := range result.Events {
					if event.Type != "structs.structs.EventPermission" && event.Type != "structs.structs.EventGuildRankPermission" {
						continue
					}
					// BaseApp adds a plain-text mode attribute outside the typed
					// JSON payload. Do not feed it to the protobuf JSON decoder.
					payload := abci.Event{Type: event.Type}
					for _, attribute := range event.Attributes {
						if attribute.Key != "mode" {
							payload.Attributes = append(payload.Attributes, attribute)
						}
					}
					parsed, err := sdk.ParseTypedEvent(payload)
					require.NoError(t, err)
					switch value := parsed.(type) {
					case *types.EventPermission:
						if strings.HasPrefix(value.PermissionRecord.PermissionId, objectID+"@") && value.PermissionRecord.Value == 0 {
							require.True(t, expectedDirect[value.PermissionRecord.PermissionId], "unexpected or duplicate direct-permission clear")
							delete(expectedDirect, value.PermissionRecord.PermissionId)
							directEvents++
						}
					case *types.EventGuildRankPermission:
						if value.GuildRankPermissionRecord.ObjectId == objectID && value.GuildRankPermissionRecord.Rank == 0 {
							require.True(t, expectedRanks[value.GuildRankPermissionRecord.GuildId], "unexpected or duplicate rank-permission clear")
							delete(expectedRanks, value.GuildRankPermissionRecord.GuildId)
							rankEvents++
						}
					}
				}
				_, err = restarted.Commit()
				require.NoError(t, err)
			}
			// Commit resets finalize-block state; read the committed check-state view.
			finalCtx := restarted.NewContextLegacy(true, cmtproto.Header{Height: height + 8, ChainID: jailGateChainID, Time: restartTime.Add(8 * time.Second)})
			require.Equal(t, uint64(1000), restarted.StructsKeeper.GetAllocationCount(finalCtx))
			require.Empty(t, restarted.StructsKeeper.GetPermissionCleanupQueue(finalCtx))
			require.Empty(t, restarted.StructsKeeper.GetPermissionsByObject(finalCtx, objectID))
			for i := 0; i < grants; i++ {
				require.Equal(t, [types.PermissionBitCount]uint64{}, restarted.StructsKeeper.ReadGuildRankRegister(finalCtx, objectID, fmt.Sprintf("0-%d", i+1)))
			}
			require.Empty(t, expectedDirect)
			require.Empty(t, expectedRanks)
			require.Equal(t, remainingDirect, directEvents)
			require.Equal(t, grants, rankEvents)
		})
	}
}
