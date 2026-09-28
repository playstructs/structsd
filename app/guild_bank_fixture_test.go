package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/x/structs/types"
)

// Bootstrap a holder position through treasury funding, issuance and distribution.
func bootstrapAppGuildTokens(t *testing.T, bApp *app.App, ms types.MsgServer, ctx sdk.Context, msg *types.MsgGuildBankMint) (*types.MsgGuildBankMintResponse, error) {
	t.Helper()
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	player, err := cc.GetPlayerByAddress(msg.Creator)
	require.NoError(t, err)
	id := msg.GuildId
	if id == "" {
		id = player.GetGuildId()
	}
	addresses, err := cc.EnsureVaults(id)
	require.NoError(t, err)
	alpha := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewIntFromUint64(msg.AmountAlpha)))
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, player.GetPrimaryAccount(), addresses[0], alpha))
	response, err := ms.GuildBankMint(ctx, msg)
	if err != nil {
		return response, err
	}
	tokens := sdk.NewCoins(sdk.NewCoin("uguild."+id, math.NewIntFromUint64(msg.AmountToken)))
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, addresses[0], player.GetPrimaryAccount(), tokens))
	return response, nil
}
