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
	owner = k.AppendPlayer(ctx, owner)

	targetPlayer := types.Player{
		Creator:        "cosmos1target",
		PrimaryAddress: "cosmos1target",
	}
	targetPlayer = k.AppendPlayer(ctx, targetPlayer)

	// Create substation and connect player
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     owner.Creator,
	}
	createdAllocation, _, err := k.AppendAllocation(ctx, allocation, 100)
	require.NoError(t, err)

	substation, _, err := k.AppendSubstation(ctx, createdAllocation, owner)
	require.NoError(t, err)

	// Connect player to substation
	connectedPlayer, err := k.SubstationConnectPlayer(ctx, substation, targetPlayer)
	require.NoError(t, err)
	require.Equal(t, substation.Id, connectedPlayer.SubstationId)

	// Grant permissions
	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(owner.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionGrid)

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
			expErrMsg: "could be found",
		},
		{
			name: "no permissions",
			input: &types.MsgSubstationPlayerDisconnect{
				Creator:  "cosmos1noperms",
				PlayerId: targetPlayer.Id,
			},
			expErr:    true,
			expErrMsg: "no Energy Management permissions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reconnect player if needed
			if tc.name == "valid player disconnection" {
				_, err := k.SubstationConnectPlayer(ctx, substation, targetPlayer)
				require.NoError(t, err)
			}

			resp, err := ms.SubstationPlayerDisconnect(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
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
