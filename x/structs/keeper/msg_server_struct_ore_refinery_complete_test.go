package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructOreRefineryComplete(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1_000_000_000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	playerAcc := sdk.AccAddress("refiner_pad_1234567890123456789012")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, sdkCtx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: player.Creator, Owner: player.Id})
	player.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, player)

	// Give the player stored ore so CanOreRefine passes
	oreAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, player.Id)
	k.SetGridAttribute(sdkCtx, oreAttrId, uint64(100))

	structType := types.StructType{
		Id:                    1,
		Type:                  "Refinery",
		Category:              types.ObjectType_planet,
		PlanetaryRefinery:     1,
		OreRefiningDifficulty: 2,
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

	blockStartAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structObj.Id)
	k.SetStructAttribute(sdkCtx, blockStartAttrId, uint64(1))

	t.Run("valid ore refinery complete", func(t *testing.T) {
		hashTemplate := structObj.Id + "REFINE1NONCE%s"
		nonce, proof := testFindProof(hashTemplate, 1)

		resp, err := ms.StructOreRefineryComplete(wctx, &types.MsgStructOreRefineryComplete{
			Creator:  player.Creator,
			StructId: structObj.Id,
			Nonce:    nonce,
			Proof:    proof,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructOreRefineryComplete(wctx, &types.MsgStructOreRefineryComplete{
			Creator:  player.Creator,
			StructId: "invalid-struct",
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructOreRefineryComplete(wctx, &types.MsgStructOreRefineryComplete{
			Creator:  "cosmos1noperms",
			StructId: structObj.Id,
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})
}
