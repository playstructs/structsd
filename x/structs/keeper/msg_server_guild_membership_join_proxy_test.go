package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgGuildMembershipJoinProxy(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	// Valid 33-byte compressed secp256k1 pubkey hex (66 chars)
	validPubKeyHex := "020000000000000000000000000000000000000000000000000000000000000001"

	t.Run("proxy not in guild", func(t *testing.T) {
		freshAcc := sdk.AccAddress("proxy_notingld_pad01")
		_, err := ms.GuildMembershipJoinProxy(wctx, &types.MsgGuildMembershipJoinProxy{
			Creator:        freshAcc.String(),
			Address:        "cosmos1wrongaddr",
			ProofPubKey:    validPubKeyHex,
			ProofSignature: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("address mismatch", func(t *testing.T) {
		_, err := ms.GuildMembershipJoinProxy(wctx, &types.MsgGuildMembershipJoinProxy{
			Creator:        gs.GuildOwner.Creator,
			Address:        "cosmos1wrongaddr",
			ProofPubKey:    validPubKeyHex,
			ProofSignature: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "proof mismatch")
	})

	t.Run("bad pubkey hex", func(t *testing.T) {
		_, err := ms.GuildMembershipJoinProxy(wctx, &types.MsgGuildMembershipJoinProxy{
			Creator:        gs.GuildOwner.Creator,
			Address:        "cosmos1wrongaddr",
			ProofPubKey:    "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			ProofSignature: "0000000000000000000000000000000000000000000000000000000000000000",
		})
		require.Error(t, err)
	})
}
