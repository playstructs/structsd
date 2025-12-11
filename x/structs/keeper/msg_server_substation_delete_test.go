package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationDelete(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Create substation
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     player.Creator,
	}
	createdAllocation, _, err := k.AppendAllocation(ctx, allocation, 100)
	require.NoError(t, err)

	substation, _, err := k.AppendSubstation(ctx, createdAllocation, player)
	require.NoError(t, err)

	// Grant permissions
	substationPermissionId := keeperlib.GetObjectPermissionIDBytes(substation.Id, player.Id)
	k.PermissionAdd(ctx, substationPermissionId, types.PermissionDelete)

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionAssets)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationDelete
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid substation deletion",
			input: &types.MsgSubstationDelete{
				Creator:               player.Creator,
				SubstationId:          substation.Id,
				MigrationSubstationId: "",
			},
			expErr: false,
		},
		{
			name: "no delete permissions",
			input: &types.MsgSubstationDelete{
				Creator:               "cosmos1noperms",
				SubstationId:          substation.Id,
				MigrationSubstationId: "",
			},
			expErr:    true,
			expErrMsg: "has no Substation Delete permissions",
		},
		{
			name: "no energy management permissions",
			input: &types.MsgSubstationDelete{
				Creator:               player.Creator,
				SubstationId:          substation.Id,
				MigrationSubstationId: "",
			},
			expErr:    true,
			expErrMsg: "no Energy Management permissions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Recreate substation if needed
			if tc.name == "valid substation deletion" {
				substation, _, _ = k.AppendSubstation(ctx, createdAllocation, player)
				k.PermissionAdd(ctx, keeperlib.GetObjectPermissionIDBytes(substation.Id, player.Id), types.PermissionDelete)
				tc.input.SubstationId = substation.Id
			} else if tc.name == "no energy management permissions" {
				k.PermissionClearAll(ctx, addressPermissionId)
			}

			resp, err := ms.SubstationDelete(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify substation was deleted
				_, found := k.GetSubstation(ctx, tc.input.SubstationId)
				require.False(t, found)
			}
		})
	}
}
