package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationAllocationDisconnect(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, ctx, player)

	// Set up source capacity
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	// Create a substation
	substationAllocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller: player.Id,
	}
	substationAlloc, err := testAppendAllocation(k, ctx, substationAllocation, 100)
	require.NoError(t, err)

	substation, _, err := testAppendSubstation(k, ctx, substationAlloc, player)
	require.NoError(t, err)

	// Create an allocation connected to substation
	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  substation.Id,
		Type:           types.AllocationType_dynamic,
		Controller: player.Id,
	}
	createdAllocation, err := testAppendAllocation(k, ctx, allocation, 100)
	require.NoError(t, err)

	// Grant permissions
	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	testPermissionAdd(k, ctx, addressPermissionId, types.PermSubstationConnection)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationAllocationDisconnect
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid allocation disconnection",
			input: &types.MsgSubstationAllocationDisconnect{
				Creator:      player.Creator,
				AllocationId: createdAllocation.Id,
			},
			expErr: false,
		},
		{
			name: "allocation not found",
			input: &types.MsgSubstationAllocationDisconnect{
				Creator:      player.Creator,
				AllocationId: "invalid-allocation",
			},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name: "no permissions",
			input: &types.MsgSubstationAllocationDisconnect{
				Creator:      "cosmos1noperms",
				AllocationId: createdAllocation.Id,
			},
			expErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Recreate connected allocation if needed
			if tc.name == "valid allocation disconnection" {
				allocation.DestinationId = substation.Id
				createdAllocation, _ = testAppendAllocation(k, ctx, allocation, 100)
				tc.input.AllocationId = createdAllocation.Id
			}

			resp, err := ms.SubstationAllocationDisconnect(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify allocation was disconnected
				updatedAllocation, found := k.GetAllocation(ctx, tc.input.AllocationId)
				require.True(t, found)
				require.Equal(t, "", updatedAllocation.DestinationId)
			}
		})
	}
}
