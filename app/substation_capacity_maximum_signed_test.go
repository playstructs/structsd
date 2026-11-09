package app_test

import (
	"math/rand"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSignedSubstationConnectionCapacityMaximum(t *testing.T) {
	for _, scenario := range []string{"owner", "delegate", "rank", "guild_owner", "missing_key_bit", "missing_object_permission", "ugc_only", "missing", "wrong_type", "closing", "later_failure"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupReactorAppGenesis(t, 1, true)
			k := bApp.StructsKeeper
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			actor := appendAppPlayer(t, k, ctx, address)
			owner := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
			station := types.Substation{Id: "4-990", Owner: owner.Id}
			bits, success := types.PermUpdate, true
			switch scenario {
			case "owner", "missing_key_bit", "closing", "later_failure", "missing", "wrong_type":
				station.Owner = actor.Id
			case "delegate":
				k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(station.Id, actor.Id), types.PermUpdate)
			case "rank", "guild_owner":
				guild := types.CreateEmptyGuild()
				guild.Id, guild.Index, guild.Owner = "0-990", 990, owner.Id
				if scenario == "guild_owner" {
					guild.Owner = actor.Id
				}
				k.SetGuild(ctx, guild)
				station.Owner = guild.Id
				if scenario == "rank" {
					actor.GuildId, actor.GuildRank = guild.Id, 3
					k.SetPlayer(ctx, actor)
					k.SetGuildRankPermissionStoreOnly(ctx, station.Id, guild.Id, types.PermUpdate, 3)
				}
			case "missing_object_permission":
				success = false
			case "ugc_only":
				bits, success = types.PermGuildUGCUpdate, false
			}
			if scenario == "missing_key_bit" {
				bits, success = types.PermPlay, false
			}
			if scenario == "closing" {
				station.DeletionPending, success = true, false
			}
			k.SetSubstation(ctx, station)
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
			capacityID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, station.Id)
			shareID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, station.Id)
			maximumID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacityMaximum, station.Id)
			k.SetGridAttribute(ctx, capacityID, 1000)
			k.SetGridAttribute(ctx, shareID, 900)
			id := station.Id
			if scenario == "missing" {
				id, success = "4-999999", false
			}
			if scenario == "wrong_type" {
				id, success = actor.Id, false
			}
			message := func(id string) sdk.Msg {
				return &types.MsgSubstationUpdateConnectionCapacityMaximum{Creator: address.String(), SubstationId: id, ConnectionCapacityMaximum: 200}
			}
			messages := []sdk.Msg{message(id)}
			if scenario == "later_failure" {
				messages, success = append(messages, message("4-999999")), false
			}
			signer := bApp.AccountKeeper.GetAccount(ctx, address)
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			current := bApp.NewContext(false)
			if success {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, uint64(200), k.GetGridAttribute(current, maximumID))
				require.Equal(t, uint64(200), k.GetGridAttribute(current, shareID))
			} else {
				require.NotZero(t, result.Code)
				require.NotContains(t, result.Log, "panic")
				require.Zero(t, k.GetGridAttribute(current, maximumID))
				require.Equal(t, uint64(900), k.GetGridAttribute(current, shareID))
				if scenario == "later_failure" {
					require.Contains(t, result.Log, "message index: 1")
				}
			}
			require.Equal(t, uint64(1000), k.GetGridAttribute(current, capacityID))
		})
	}
}
