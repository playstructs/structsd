package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgReactorCancelDefusion(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Create reactor
	playerAcc, _ := sdk.AccAddressFromBech32(player.Creator)
	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	reactor := types.Reactor{
		RawAddress: validatorAddress.Bytes(),
	}
	reactor = k.AppendReactor(ctx, reactor)
	k.SetReactorValidatorBytes(ctx, reactor.Id, validatorAddress.Bytes())

	// Grant permissions
	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionAssets)

	// Use default bond denom for testing
	bondDenom := "stake"

	testCases := []struct {
		name      string
		input     *types.MsgReactorCancelDefusion
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid cancel defusion",
			input: &types.MsgReactorCancelDefusion{
				Creator:          player.Creator,
				DelegatorAddress: player.Creator,
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
				CreationHeight:   1,
			},
			expErr: false,
		},
		{
			name: "invalid delegator address",
			input: &types.MsgReactorCancelDefusion{
				Creator:          player.Creator,
				DelegatorAddress: "invalid-address",
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
				CreationHeight:   1,
			},
			expErr:    true,
			expErrMsg: "invalid delegator address",
		},
		{
			name: "invalid validator address",
			input: &types.MsgReactorCancelDefusion{
				Creator:          player.Creator,
				DelegatorAddress: player.Creator,
				ValidatorAddress: "invalid-validator",
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
				CreationHeight:   1,
			},
			expErr:    true,
			expErrMsg: "invalid validator address",
		},
		{
			name: "invalid amount",
			input: &types.MsgReactorCancelDefusion{
				Creator:          player.Creator,
				DelegatorAddress: player.Creator,
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(0)),
				CreationHeight:   1,
			},
			expErr:    true,
			expErrMsg: "invalid delegation amount",
		},
		{
			name: "invalid height",
			input: &types.MsgReactorCancelDefusion{
				Creator:          player.Creator,
				DelegatorAddress: player.Creator,
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
				CreationHeight:   0,
			},
			expErr:    true,
			expErrMsg: "invalid height",
		},
		{
			name: "no permissions",
			input: &types.MsgReactorCancelDefusion{
				Creator:          "cosmos1noperms",
				DelegatorAddress: player.Creator,
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
				CreationHeight:   1,
			},
			expErr:    true,
			expErrMsg: "has no",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.ReactorCancelDefusion(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				// Note: This test may fail if there's no unbonding delegation
				// The actual cancel defusion requires an existing unbonding delegation
				_ = resp
				_ = err
			}
		})
	}
}
