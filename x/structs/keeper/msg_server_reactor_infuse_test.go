package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgReactorInfuse(t *testing.T) {
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

	// Set up balances
	// Use default bond denom for testing
	bondDenom := "stake"
	coins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1000)))
	k.BankKeeper().MintCoins(ctx, types.ModuleName, coins)
	k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, coins)

	testCases := []struct {
		name      string
		input     *types.MsgReactorInfuse
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid reactor infuse",
			input: &types.MsgReactorInfuse{
				Creator:          player.Creator,
				DelegatorAddress: player.Creator,
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
			},
			expErr: false,
		},
		{
			name: "invalid delegator address",
			input: &types.MsgReactorInfuse{
				Creator:          player.Creator,
				DelegatorAddress: "invalid-address",
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
			},
			expErr:    true,
			expErrMsg: "invalid delegator address",
		},
		{
			name: "invalid validator address",
			input: &types.MsgReactorInfuse{
				Creator:          player.Creator,
				DelegatorAddress: player.Creator,
				ValidatorAddress: "invalid-validator",
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
			},
			expErr:    true,
			expErrMsg: "invalid validator address",
		},
		{
			name: "invalid amount",
			input: &types.MsgReactorInfuse{
				Creator:          player.Creator,
				DelegatorAddress: player.Creator,
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(0)),
			},
			expErr:    true,
			expErrMsg: "invalid delegation amount",
		},
		{
			name: "no permissions",
			input: &types.MsgReactorInfuse{
				Creator:          "cosmos1noperms",
				DelegatorAddress: player.Creator,
				ValidatorAddress: reactor.Validator,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(100)),
			},
			expErr:    true,
			expErrMsg: "has no",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.ReactorInfuse(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}
