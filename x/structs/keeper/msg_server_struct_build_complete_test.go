package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructBuildComplete(t *testing.T) {
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

	structType := types.StructType{
		Id:              1,
		Type:            "Test Build",
		Category:        types.ObjectType_planet,
		BuildDraw:       100,
		PassiveDraw:     50,
		BuildDifficulty: 2,
	}
	k.SetStructType(sdkCtx, structType)

	structObj := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	structObj = testAppendStruct(k, sdkCtx, structObj)

	blockStartAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structObj.Id)
	k.SetStructAttribute(sdkCtx, blockStartAttrId, uint64(1))

	t.Run("valid build complete", func(t *testing.T) {
		hashTemplate := structObj.Id + "BUILD1NONCE%s"
		nonce, proof := testFindProof(hashTemplate, 1)

		resp, err := ms.StructBuildComplete(wctx, &types.MsgStructBuildComplete{
			Creator:  player.Creator,
			StructId: structObj.Id,
			Nonce:    nonce,
			Proof:    proof,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructBuildComplete(wctx, &types.MsgStructBuildComplete{
			Creator:  player.Creator,
			StructId: "invalid-struct",
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})

	t.Run("struct already built", func(t *testing.T) {
		_, err := ms.StructBuildComplete(wctx, &types.MsgStructBuildComplete{
			Creator:  player.Creator,
			StructId: structObj.Id,
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "built")
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructBuildComplete(wctx, &types.MsgStructBuildComplete{
			Creator:  "cosmos1noperms",
			StructId: structObj.Id,
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})
}
