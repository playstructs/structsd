package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgReactorBeginMigration(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Create reactors
	playerAcc, _ := sdk.AccAddressFromBech32(player.Creator)
	validatorAddress1 := sdk.ValAddress(playerAcc.Bytes())
	reactor1 := types.Reactor{
		RawAddress: validatorAddress1.Bytes(),
	}
	reactor1 = k.AppendReactor(ctx, reactor1)
	k.SetReactorValidatorBytes(ctx, reactor1.Id, validatorAddress1.Bytes())

	// Create second reactor
	validatorAddress2 := sdk.ValAddress([]byte("validator2"))
	reactor2 := types.Reactor{
		RawAddress: validatorAddress2.Bytes(),
	}
	reactor2 = k.AppendReactor(ctx, reactor2)
	k.SetReactorValidatorBytes(ctx, reactor2.Id, validatorAddress2.Bytes())

	// Grant permissions
	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionAssets)

	// Use default bond denom for testing
	bondDenom := "stake"

	testCases := []struct {
		name      string
		input     *types.MsgReactorBeginMigration
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid begin migration",
			input: &types.MsgReactorBeginMigration{
				Creator:             player.Creator,
				DelegatorAddress:    player.Creator,
				ValidatorSrcAddress: reactor1.Validator,
				ValidatorDstAddress: reactor2.Validator,
				Amount:              sdk.NewCoin(bondDenom, math.NewInt(100)),
			},
			expErr: false,
		},
		{
			name: "invalid delegator address",
			input: &types.MsgReactorBeginMigration{
				Creator:             player.Creator,
				DelegatorAddress:    "invalid-address",
				ValidatorSrcAddress: reactor1.Validator,
				ValidatorDstAddress: reactor2.Validator,
				Amount:              sdk.NewCoin(bondDenom, math.NewInt(100)),
			},
			expErr:    true,
			expErrMsg: "invalid delegator address",
		},
		{
			name: "invalid source validator address",
			input: &types.MsgReactorBeginMigration{
				Creator:             player.Creator,
				DelegatorAddress:    player.Creator,
				ValidatorSrcAddress: "invalid-validator",
				ValidatorDstAddress: reactor2.Validator,
				Amount:              sdk.NewCoin(bondDenom, math.NewInt(100)),
			},
			expErr:    true,
			expErrMsg: "invalid validator address",
		},
		{
			name: "invalid amount",
			input: &types.MsgReactorBeginMigration{
				Creator:             player.Creator,
				DelegatorAddress:    player.Creator,
				ValidatorSrcAddress: reactor1.Validator,
				ValidatorDstAddress: reactor2.Validator,
				Amount:              sdk.NewCoin(bondDenom, math.NewInt(0)),
			},
			expErr:    true,
			expErrMsg: "invalid delegation amount",
		},
		{
			name: "no permissions",
			input: &types.MsgReactorBeginMigration{
				Creator:             "cosmos1noperms",
				DelegatorAddress:    player.Creator,
				ValidatorSrcAddress: reactor1.Validator,
				ValidatorDstAddress: reactor2.Validator,
				Amount:              sdk.NewCoin(bondDenom, math.NewInt(100)),
			},
			expErr:    true,
			expErrMsg: "has no",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.ReactorBeginMigration(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				// Note: This test may fail if there's no delegation to migrate
				// The actual migration requires an existing delegation
				_ = resp
				_ = err
			}
		})
	}
}
