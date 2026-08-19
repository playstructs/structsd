package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructOreMinerComplete(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1_000_000_000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, sdkCtx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: player.Creator, Owner: player.Id})
	player.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, player)

	structType := types.StructType{
		Id:                  1,
		Type:                "Miner",
		Category:            types.ObjectType_planet,
		PlanetaryMining:     1,
		OreMiningDifficulty: 2,
	}
	k.SetStructType(sdkCtx, structType)

	structObj := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   planet.Id,
		LocationType: types.ObjectType_planet,
	}
	structObj = testAppendStruct(k, sdkCtx, structObj)

	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateOnline))

	mineClockAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreMine, planet.Id)
	k.SetPlanetAttribute(sdkCtx, mineClockAttrId, uint64(1))
	mineQtyAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreMiningActiveQuantity, planet.Id)
	k.SetPlanetAttribute(sdkCtx, mineQtyAttrId, uint64(1))

	t.Run("valid ore miner complete", func(t *testing.T) {
		hashTemplate := structObj.Id + "MINE1NONCE%s"
		nonce, proof := testFindProof(hashTemplate, 1)

		resp, err := ms.StructOreMinerComplete(wctx, &types.MsgStructOreMinerComplete{
			Creator:  player.Creator,
			StructId: structObj.Id,
			Nonce:    nonce,
			Proof:    proof,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// A successful extraction re-anchors the shared planet clock to the
		// current height so the next round starts from zero age, and leaves
		// the active-rig counter untouched.
		require.Equal(t, uint64(sdkCtx.BlockHeight()), k.GetPlanetAttribute(sdkCtx, mineClockAttrId))
		require.Equal(t, uint64(1), k.GetPlanetAttribute(sdkCtx, mineQtyAttrId))
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructOreMinerComplete(wctx, &types.MsgStructOreMinerComplete{
			Creator:  player.Creator,
			StructId: "invalid-struct",
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructOreMinerComplete(wctx, &types.MsgStructOreMinerComplete{
			Creator:  "cosmos1noperms",
			StructId: structObj.Id,
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})
}
