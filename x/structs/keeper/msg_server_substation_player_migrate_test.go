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
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	wctx := sdk.WrapSDKContext(sdkCtx)

	owner := types.Player{
		Creator:        "cosmos1owner",
		PrimaryAddress: "cosmos1owner",
	}
	owner = testAppendPlayer(k, sdkCtx, owner)

	targetPlayer := types.Player{
		Creator:        "cosmos1target",
		PrimaryAddress: "cosmos1target",
	}
	targetPlayer = testAppendPlayer(k, sdkCtx, targetPlayer)

	reactor := testAppendReactor(k, sdkCtx, types.Reactor{Validator: "cosmosvaloper1test"})
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactor.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(10000))

	allocation := types.Allocation{
		SourceObjectId: reactor.Id,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     owner.Id,
	}
	createdAllocation, err := testAppendAllocation(k, sdkCtx, allocation, 100)
	require.NoError(t, err)

	substation, _, err := testAppendSubstation(k, sdkCtx, createdAllocation, owner)
	require.NoError(t, err)

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(owner.Creator)
	testPermissionAdd(k, sdkCtx, addressPermissionId, types.PermAll)

	substationPermissionId := keeperlib.GetObjectPermissionIDBytes(substation.Id, owner.Id)
	testPermissionAdd(k, sdkCtx, substationPermissionId, types.PermAll)

	targetPlayerPermissionId := keeperlib.GetObjectPermissionIDBytes(targetPlayer.Id, owner.Id)
	testPermissionAdd(k, sdkCtx, targetPlayerPermissionId, types.PermAll)

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
			expErrMsg: "not found",
		},
		{
			name: "target player not found",
			input: &types.MsgSubstationPlayerMigrate{
				Creator:      owner.Creator,
				SubstationId: substation.Id,
				PlayerId:     []string{"invalid-player"},
			},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name: "no substation permissions",
			input: &types.MsgSubstationPlayerMigrate{
				Creator:      "cosmos1noperms",
				SubstationId: substation.Id,
				PlayerId:     []string{targetPlayer.Id},
			},
			expErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.SubstationPlayerMigrate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}
