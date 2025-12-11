package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationCreate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Set up source capacity for allocation
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	// Grant permissions
	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionAssets)

	// Create an allocation
	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     player.Creator,
	}
	createdAllocation, _, err := k.AppendAllocation(ctx, allocation, 100)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationCreate
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid substation creation",
			input: &types.MsgSubstationCreate{
				Creator:      player.Creator,
				AllocationId: createdAllocation.Id,
			},
			expErr: false,
		},
		{
			name: "allocation not found",
			input: &types.MsgSubstationCreate{
				Creator:      player.Creator,
				AllocationId: "invalid-allocation",
			},
			expErr:    true,
			expErrMsg: "allocation not found",
		},
		{
			name: "no energy management permissions",
			input: &types.MsgSubstationCreate{
				Creator:      "cosmos1noperms",
				AllocationId: createdAllocation.Id,
			},
			expErr:    true,
			expErrMsg: "no Energy Management permissions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.SubstationCreate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.SubstationId)

				// Verify substation was created
				substation, found := k.GetSubstation(ctx, resp.SubstationId)
				require.True(t, found)
				require.Equal(t, player.Id, substation.Owner)
			}
		})
	}
}
