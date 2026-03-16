package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgPlayerSend(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	toAcc := sdk.AccAddress("toaddress123456789012345678901234567890")

	t.Run("valid send", func(t *testing.T) {
		coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1000)))
		k.BankKeeper().MintCoins(ctx, types.ModuleName, coins)
		k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, coins)

		_, err := ms.PlayerSend(wctx, &types.MsgPlayerSend{
			Creator:     player.Creator,
			FromAddress: player.Creator,
			ToAddress:   toAcc.String(),
			Amount:      sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(100))),
		})
		require.NoError(t, err)
	})

	t.Run("unregistered from address", func(t *testing.T) {
		unregistered := sdk.AccAddress("unregistered12345678901234567890123").String()
		_, err := ms.PlayerSend(wctx, &types.MsgPlayerSend{
			Creator:     player.Creator,
			FromAddress: unregistered,
			ToAddress:   toAcc.String(),
			Amount:      sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(100))),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not associated")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregistered := sdk.AccAddress("unregistered12345678901234567890123").String()

		resp, err := ms.PlayerSend(wctx, &types.MsgPlayerSend{
			Creator:     unregistered,
			FromAddress: player.Creator,
			ToAddress:   toAcc.String(),
			Amount:      sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(100))),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not associated")
		_ = resp
	})
}
