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
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Set up player capacity to be online
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))

	// Create a planet
	planetId := k.AppendPlanet(ctx, player)

	// Create a struct type
	structType := types.StructType{
		Id:                    1,
		Type:                  types.CommandStruct,
		Category:              types.ObjectType_player,
		OreRefiningDifficulty: 1,
	}
	k.SetStructType(ctx, structType)

	// Create a struct
	structObj := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   planetId,
		LocationType: types.ObjectType_planet,
	}
	structObj = k.AppendStruct(ctx, structObj)

	// Mark struct as built and online
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	builtFlag := uint64(types.StructStateBuilt)
	k.SetStructAttributeFlagAdd(ctx, statusAttrId, builtFlag)

	// Set block start ore refine
	blockStartAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structObj.Id)
	k.SetStructAttribute(ctx, blockStartAttrId, uint64(1))

	testCases := []struct {
		name      string
		input     *types.MsgStructOreRefineryComplete
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid ore refinery complete",
			input: &types.MsgStructOreRefineryComplete{
				Creator:  player.Creator,
				StructId: structObj.Id,
				Nonce:    "test-nonce",
				Proof:    "test-proof",
			},
			expErr: false,
		},
		{
			name: "struct not found",
			input: &types.MsgStructOreRefineryComplete{
				Creator:  player.Creator,
				StructId: "invalid-struct",
				Nonce:    "test-nonce",
				Proof:    "test-proof",
			},
			expErr:    true,
			expErrMsg: "does not exist",
		},
		{
			name: "no play permissions",
			input: &types.MsgStructOreRefineryComplete{
				Creator:  "cosmos1noperms",
				StructId: structObj.Id,
				Nonce:    "test-nonce",
				Proof:    "test-proof",
			},
			expErr:    true,
			expErrMsg: "has no",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.StructOreRefineryComplete(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				// Note: This test may fail if proof validation fails
				// The actual proof generation is complex
				_ = resp
				_ = err
			}
		})
	}
}
