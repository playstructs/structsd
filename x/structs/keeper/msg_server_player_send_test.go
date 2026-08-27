package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
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

/* TestPlayerSendHonoursBankPolicy is the regression on a module handler routing
 * around the bank's own rules.
 *
 * BaseSendKeeper.SendCoins is not the bank's policy layer. It validates the coin
 * structure, applies any registered send restriction, and moves balances - the
 * blocked-address set and the per-denom send-enabled flags live in the bank's
 * MsgServer, which a module calling SendCoins directly never reaches. So the app
 * could declare the staking pools and the fee collector unreachable and this
 * handler would reach them anyway, and governance could freeze a denom and this
 * handler would still move it.
 *
 * The staking pools are the sharp end: coins arriving there with no matching
 * delegation leave the pool balance disagreeing with staking's recorded tokens,
 * which surfaces as a panic on the next export-and-restart rather than as a
 * failed transaction.
 */
func TestPlayerSendHonoursBankPolicy(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	senderAcc := sdk.AccAddress("bankpolicy_sender_pad_addr_000001")
	sender := testAppendPlayer(k, ctx, types.Player{
		Creator:        senderAcc.String(),
		PrimaryAddress: senderAcc.String(),
	})

	mockBank := k.BankKeeper().(*keepertest.MockBankKeeper)

	fund := func(denom string, amount int64) {
		coins := sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(amount)))
		require.NoError(t, mockBank.MintCoins(ctx, types.ModuleName, coins))
		require.NoError(t, mockBank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, senderAcc, coins))
	}
	fund("ualpha", 10000)

	t.Run("a blocked module account is not a valid recipient", func(t *testing.T) {
		// Stand-in for the bonded staking pool, which the app declares
		// unreachable via BlockedModuleAccountsOverride.
		bondedPool := sdk.AccAddress("bankpolicy_bonded_pool_addr_0001")
		mockBank.BlockAddress(bondedPool)

		_, err := ms.PlayerSend(wctx, &types.MsgPlayerSend{
			Creator:     sender.Creator,
			FromAddress: sender.Creator,
			ToAddress:   bondedPool.String(),
			Amount:      sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(100))),
		})
		require.Error(t, err, "an account the app declared unreachable must stay unreachable")
		require.Contains(t, err.Error(), "blocked_recipient")

		require.True(t, mockBank.SpendableCoins(ctx, bondedPool).IsZero(),
			"and nothing may have landed there")
	})

	t.Run("a send-disabled denom cannot move", func(t *testing.T) {
		fund("ufrozen", 5000)
		mockBank.DisableSendDenom("ufrozen")

		recipient := sdk.AccAddress("bankpolicy_recipient_addr_00001")
		_, err := ms.PlayerSend(wctx, &types.MsgPlayerSend{
			Creator:     sender.Creator,
			FromAddress: sender.Creator,
			ToAddress:   recipient.String(),
			Amount:      sdk.NewCoins(sdk.NewCoin("ufrozen", math.NewInt(100))),
		})
		require.Error(t, err, "a governance freeze must apply here too")

		require.True(t, mockBank.SpendableCoins(ctx, recipient).IsZero())
	})

	t.Run("a malformed recipient is rejected rather than becoming the empty address", func(t *testing.T) {
		_, err := ms.PlayerSend(wctx, &types.MsgPlayerSend{
			Creator:     sender.Creator,
			FromAddress: sender.Creator,
			ToAddress:   "not-a-bech32-address",
			Amount:      sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(100))),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "couldn't be validated as a real address")
	})

	// The control: an ordinary recipient and an ordinary denom still move, so
	// the refusals above are about policy and not about the handler.
	t.Run("an ordinary send is unaffected", func(t *testing.T) {
		recipient := sdk.AccAddress("bankpolicy_ordinary_addr_000001")

		_, err := ms.PlayerSend(wctx, &types.MsgPlayerSend{
			Creator:     sender.Creator,
			FromAddress: sender.Creator,
			ToAddress:   recipient.String(),
			Amount:      sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(100))),
		})
		require.NoError(t, err)
		require.Equal(t, math.NewInt(100), mockBank.SpendableCoin(ctx, recipient, "ualpha").Amount)
	})
}
