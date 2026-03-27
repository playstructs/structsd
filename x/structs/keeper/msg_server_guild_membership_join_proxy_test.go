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

	// KeeperStartValue=1, so first guild is 0-1 (matches signed test data)
	gs := testCreateGuild(k, ctx)

	// Real TX data: signature was computed over "GUILD0-1ADDRESSstructs1sju8kv398dfraagl2fdfjn7km9h2mykra08ndnNONCE0"
	const (
		proxyTargetAddress = "structs1sju8kv398dfraagl2fdfjn7km9h2mykra08ndn"
		proxyPubKeyHex     = "027e07e1ac6dfbe5a20d8f7dd401563d344c1fd75a05a33f3223f24ef702947e56"
		proxySignatureHex  = "1d4f37c287e467c9b451c91f047d816b70ff2ea4827027e186d46c57f032b4de5f3e3f1d13aeb086a6d16e237a3df20f98826f0471e4c2a7de1e856eebba9d3401"
	)

	// Valid 33-byte compressed secp256k1 pubkey hex (66 chars)
	validPubKeyHex := "020000000000000000000000000000000000000000000000000000000000000001"

	t.Run("valid proxy join", func(t *testing.T) {
		require.Equal(t, "0-1", gs.Guild.Id, "guild ID must be 0-1 to match signed data")

		resp, err := ms.GuildMembershipJoinProxy(wctx, &types.MsgGuildMembershipJoinProxy{
			Creator:        gs.GuildOwner.Creator,
			Address:        proxyTargetAddress,
			ProofPubKey:    proxyPubKeyHex,
			ProofSignature: proxySignatureHex,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		playerIndex := k.GetPlayerIndexFromAddress(ctx, proxyTargetAddress)
		require.NotEqual(t, uint64(0), playerIndex, "target should have a player account")

		playerId := GetObjectID(types.ObjectType_player, playerIndex)
		player, found := k.GetPlayer(ctx, playerId)
		require.True(t, found)
		require.Equal(t, gs.Guild.Id, player.GuildId, "target player should be in the proxy guild")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		freshAcc := sdk.AccAddress("proxy_notingld_pad01")
		_, err := ms.GuildMembershipJoinProxy(wctx, &types.MsgGuildMembershipJoinProxy{
			Creator:        freshAcc.String(),
			Address:        "cosmos1wrongaddr",
			ProofPubKey:    validPubKeyHex,
			ProofSignature: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not associated with a player")
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
