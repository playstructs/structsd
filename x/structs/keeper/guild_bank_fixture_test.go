package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// bootstrapGuildTokens funds treasury from a test player's existing alpha,
// performs the real mint, then distributes the issued tokens to that player.
// Conversion/redeem tests need holder balances, while issuance itself now always
// credits the guild vault. This fixture does not stand in for mint routing tests.
func bootstrapGuildTokens(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, msg *types.MsgGuildBankMint) (*types.MsgGuildBankMintResponse, error) {
	return bootstrapGuildTokenPosition(t, k, ms, ctx, msg, false)
}

func bootstrapGuildVaultTokens(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, msg *types.MsgGuildBankMint) (*types.MsgGuildBankMintResponse, error) {
	return bootstrapGuildTokenPosition(t, k, ms, ctx, msg, true)
}

func bootstrapGuildTokenPosition(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, msg *types.MsgGuildBankMint, useVault bool) (*types.MsgGuildBankMintResponse, error) {
	t.Helper()
	cc := k.NewCurrentContext(ctx)
	player, err := cc.GetPlayerByAddress(msg.Creator)
	require.NoError(t, err)
	id := msg.GuildId
	if id == "" {
		id = player.GetGuildId()
	}
	addresses, err := cc.EnsureVaults(id, player.ID())
	require.NoError(t, err)
	account := player.GetPrimaryAccount()
	if useVault {
		account = addresses[1]
	}
	alpha := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewIntFromUint64(msg.AmountAlpha)))
	require.NoError(t, k.BankKeeper().SendCoins(ctx, account, addresses[0], alpha))
	response, err := ms.GuildBankMint(ctx, msg)
	if err != nil {
		return response, err
	}
	tokens := sdk.NewCoins(sdk.NewCoin("uguild."+id, math.NewIntFromUint64(msg.AmountToken)))
	require.NoError(t, k.BankKeeper().SendCoins(ctx, addresses[0], account, tokens))
	return response, nil
}
