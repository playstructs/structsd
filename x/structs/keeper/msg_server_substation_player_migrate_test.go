package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationPlayerMigrate(t *testing.T) {
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

	// Create substation
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

	// Grant permissions
	substationPermissionId := keeperlib.GetObjectPermissionIDBytes(substation.Id, owner.Id)
	k.PermissionAdd(ctx, substationPermissionId, types.PermissionGrid)

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(owner.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionGrid)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationPlayerMigrate
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid player migration",
			input: &types.MsgSubstationPlayerMigrate{
				Creator:      owner.Creator,
				SubstationId: substation.Id,
				PlayerId:     []string{targetPlayer.Id},
			},
			expErr: false,
		},
		{
			name: "substation not found",
			input: &types.MsgSubstationPlayerMigrate{
				Creator:      owner.Creator,
				SubstationId: "invalid-substation",
				PlayerId:     []string{targetPlayer.Id},
			},
			expErr:    true,
			expErrMsg: "substation not found",
		},
		{
			name: "target player not found",
			input: &types.MsgSubstationPlayerMigrate{
				Creator:      owner.Creator,
				SubstationId: substation.Id,
				PlayerId:     []string{"invalid-player"},
			},
			expErr:    true,
			expErrMsg: "could be be found",
		},
		{
			name: "no substation permissions",
			input: &types.MsgSubstationPlayerMigrate{
				Creator:      "cosmos1noperms",
				SubstationId: substation.Id,
				PlayerId:     []string{targetPlayer.Id},
			},
			expErr:    true,
			expErrMsg: "no Energy Management permissions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.SubstationPlayerMigrate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify player was migrated
				updatedPlayer, found := k.GetPlayer(ctx, targetPlayer.Id)
				require.True(t, found)
				require.Equal(t, substation.Id, updatedPlayer.SubstationId)
			}
		})
	}
}
