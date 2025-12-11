package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructStealthDeactivate(t *testing.T) {
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

	// Set last action to ensure player has charge
	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

	// Create a struct type with stealth system
	// Note: HasStealthSystem is a method, not a field
	structType := types.StructType{
		Id:                    1,
		Type:                  types.CommandStruct,
		Category:              types.ObjectType_player,
		StealthActivateCharge: 10,
	}
	k.SetStructType(ctx, structType)

	// Create a struct
	structObj := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	structObj = k.AppendStruct(ctx, structObj)

	// Mark struct as built and online
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	builtFlag := uint64(types.StructStateBuilt)
	k.SetStructAttributeFlagAdd(ctx, statusAttrId, builtFlag)

	testCases := []struct {
		name      string
		input     *types.MsgStructStealthDeactivate
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid stealth deactivation",
			input: &types.MsgStructStealthDeactivate{
				Creator:  player.Creator,
				StructId: structObj.Id,
			},
			expErr: false,
		},
		{
			name: "struct not found",
			input: &types.MsgStructStealthDeactivate{
				Creator:  player.Creator,
				StructId: "invalid-struct",
			},
			expErr:    true,
			expErrMsg: "does not exist",
		},
		{
			name: "not in stealth",
			input: &types.MsgStructStealthDeactivate{
				Creator:  player.Creator,
				StructId: structObj.Id,
			},
			expErr:    true,
			expErrMsg: "already in out of stealth",
		},
		{
			name: "no stealth system",
			input: &types.MsgStructStealthDeactivate{
				Creator:  player.Creator,
				StructId: structObj.Id,
			},
			expErr:    true,
			expErrMsg: "has no stealth system",
		},
		{
			name: "no play permissions",
			input: &types.MsgStructStealthDeactivate{
				Creator:  "cosmos1noperms",
				StructId: structObj.Id,
			},
			expErr:    true,
			expErrMsg: "has no",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up struct state for each test
			if tc.name == "valid stealth deactivation" {
				// Ensure struct is hidden
				hiddenFlag := uint64(types.StructStateHidden)
				k.SetStructAttributeFlagAdd(ctx, statusAttrId, hiddenFlag)
			} else if tc.name == "not in stealth" {
				// Ensure struct is not hidden
				hiddenFlag := uint64(types.StructStateHidden)
				k.SetStructAttributeFlagRemove(ctx, statusAttrId, hiddenFlag)
			} else if tc.name == "no stealth system" {
				// Create struct type without stealth
				// Note: This test may not work if all struct types have stealth
				// The actual check is done via HasStealthSystem() method
				noStealthType := types.StructType{
					Id:       2,
					Type:     types.CommandStruct,
					Category: types.ObjectType_player,
				}
				k.SetStructType(ctx, noStealthType)
				structObj.Type = noStealthType.Id
				k.SetStruct(ctx, structObj)
			}

			resp, err := ms.StructStealthDeactivate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.Struct)
			}
		})
	}
}
