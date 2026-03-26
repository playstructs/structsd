package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationPlayerDisconnect(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create players
	owner := types.Player{
		Creator:        "cosmos1owner",
		PrimaryAddress: "cosmos1owner",
	}
	owner = testAppendPlayer(k, ctx, owner)

	targetPlayer := types.Player{
		Creator:        "cosmos1target",
		PrimaryAddress: "cosmos1target",
	}
	targetPlayer = testAppendPlayer(k, ctx, targetPlayer)

	// Create substation and connect player
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller: owner.Id,
	}
	createdAllocation, err := testAppendAllocation(k, ctx, allocation, 100)
	require.NoError(t, err)

	substation, _, err := testAppendSubstation(k, ctx, createdAllocation, owner)
	require.NoError(t, err)

	// Connect player to substation by setting SubstationId directly
	targetPlayer.SubstationId = substation.Id
	k.SetPlayer(ctx, targetPlayer)

	// Grant permissions
	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(owner.Creator)
	testPermissionAdd(k, ctx, addressPermissionId, types.PermSubstationConnection)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationPlayerDisconnect
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid player disconnection",
			input: &types.MsgSubstationPlayerDisconnect{
				Creator:  owner.Creator,
				PlayerId: targetPlayer.Id,
			},
			expErr: false,
		},
		{
			name: "target player not found",
			input: &types.MsgSubstationPlayerDisconnect{
				Creator:  owner.Creator,
				PlayerId: "invalid-player",
			},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name: "no permissions",
			input: &types.MsgSubstationPlayerDisconnect{
				Creator:  "cosmos1noperms",
				PlayerId: targetPlayer.Id,
			},
			expErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reconnect player if needed
			if tc.name == "valid player disconnection" {
				targetPlayer.SubstationId = substation.Id
				k.SetPlayer(ctx, targetPlayer)
			}

			resp, err := ms.SubstationPlayerDisconnect(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify player was disconnected
				updatedPlayer, found := k.GetPlayer(ctx, targetPlayer.Id)
				require.True(t, found)
				require.Equal(t, "", updatedPlayer.SubstationId)
			}
		})
	}
}
