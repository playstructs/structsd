package keeper_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
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
			Creator:                         gs.GuildOwner.Creator,
			Address:                         proxyTargetAddress,
			ProofPubKey:                     proxyPubKeyHex,
			ProofSignature:                  proxySignatureHex,
			PlayerPfpClientRenderAttributes: "{\n  \"head\": 1234\n}",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		playerIndex := k.GetPlayerIndexFromAddress(ctx, proxyTargetAddress)
		require.NotEqual(t, uint64(0), playerIndex, "target should have a player account")

		playerId := GetObjectID(types.ObjectType_player, playerIndex)
		player, found := k.GetPlayer(ctx, playerId)
		require.True(t, found)
		require.Equal(t, gs.Guild.Id, player.GuildId, "target player should be in the proxy guild")
		require.Equal(t, `{"head":1234}`, player.PfpClientRenderAttributes, "render attributes should be stored compacted")
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

	t.Run("short signature is rejected without panicking", func(t *testing.T) {
		_, err := ms.GuildMembershipJoinProxy(wctx, &types.MsgGuildMembershipJoinProxy{
			Creator:        gs.GuildOwner.Creator,
			Address:        proxyTargetAddress,
			ProofPubKey:    proxyPubKeyHex,
			ProofSignature: "00",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "signature_invalid_length")
	})
}

func TestGuildMembershipJoinProxyRefusesExistingMember(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	gs := testCreateGuild(k, ctx)

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey().Bytes()
	address := types.PubKeyToBech32(pubKey)
	member := testAppendPlayer(k, ctx, types.Player{
		Creator:        address,
		PrimaryAddress: address,
		GuildId:        "4-other",
		GuildRank:      7,
	})

	hashInput := fmt.Sprintf("GUILD%sADDRESS%sNONCE%d", gs.Guild.Id, address, 0)
	signature, err := privKey.Sign([]byte(hashInput))
	require.NoError(t, err)

	_, err = ms.GuildMembershipJoinProxy(ctx, &types.MsgGuildMembershipJoinProxy{
		Creator:        gs.GuildOwner.Creator,
		Address:        address,
		ProofPubKey:    hex.EncodeToString(pubKey),
		ProofSignature: hex.EncodeToString(signature),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already a member")

	after, found := k.GetPlayer(ctx, member.Id)
	require.True(t, found)
	require.Equal(t, "4-other", after.GuildId)
	require.Equal(t, uint64(7), after.GuildRank)
}

/* TestGuildMembershipJoinProxyRequiresMembershipBitOnRegisteredAddress is the
 * regression on the proxy path being a way around a restricted key.
 *
 * UpsertPlayer is an in-get, not an insert: for an address already bound to a
 * player it returns that whole player, and the handler then sets its guild, its
 * rank, its substation connection and its profile. The direct join path demands
 * PermGuildMembership of the acting address before any of that, and restricted
 * secondary addresses exist so a low-trust key can play without being able to
 * move the player between guilds. Key possession was being treated as enough
 * here, so such a key could do through the proxy exactly what it is barred from
 * doing directly.
 *
 * The onboarding case has to keep working, so the two are asserted together:
 * an address nobody has registered is still joined without any permission grant.
 */
func TestGuildMembershipJoinProxyRequiresMembershipBitOnRegisteredAddress(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	gs := testCreateGuild(k, ctx)

	proxyJoin := func(privKey *secp256k1.PrivKey, address string, nonce uint64) error {
		hashInput := fmt.Sprintf("GUILD%sADDRESS%sNONCE%d", gs.Guild.Id, address, nonce)
		signature, err := privKey.Sign([]byte(hashInput))
		require.NoError(t, err)

		_, err = ms.GuildMembershipJoinProxy(ctx, &types.MsgGuildMembershipJoinProxy{
			Creator:        gs.GuildOwner.Creator,
			Address:        address,
			ProofPubKey:    hex.EncodeToString(privKey.PubKey().Bytes()),
			ProofSignature: hex.EncodeToString(signature),
		})
		return err
	}

	t.Run("an unregistered address is still onboarded permissionlessly", func(t *testing.T) {
		privKey := secp256k1.GenPrivKey()
		address := types.PubKeyToBech32(privKey.PubKey().Bytes())

		require.NoError(t, proxyJoin(privKey, address, 0))

		index := k.GetPlayerIndexFromAddress(ctx, address)
		require.NotZero(t, index, "the proxy join should have created a player")
		onboarded, found := k.GetPlayer(ctx, keeperlib.GetObjectID(types.ObjectType_player, index))
		require.True(t, found)
		require.Equal(t, gs.Guild.Id, onboarded.GuildId)
	})

	t.Run("a restricted registered address cannot join its player", func(t *testing.T) {
		// A guildless player whose primary key holds everything.
		ownerAcc := sdk.AccAddress("proxy_perm_primary01")
		victim := testAppendPlayer(k, ctx, types.Player{
			Creator:        ownerAcc.String(),
			PrimaryAddress: ownerAcc.String(),
		})
		require.Empty(t, victim.GuildId)

		// A secondary key of that player, deliberately scoped to play and
		// nothing else - exactly what the direct join path would refuse.
		privKey := secp256k1.GenPrivKey()
		limited := types.PubKeyToBech32(privKey.PubKey().Bytes())
		// The index row is written regardless; the returned error is only about
		// provisioning an auth account, which the test env's bech32 prefix
		// prevents for a PubKeyToBech32 address. testAppendPlayer ignores it for
		// the same reason.
		_ = k.SetPlayerIndexForAddress(ctx, limited, victim.Index)
		require.Equal(t, victim.Index, k.GetPlayerIndexFromAddress(ctx, limited))
		k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(limited), types.PermPlay)

		err := proxyJoin(privKey, limited, 0)
		require.Error(t, err, "a key without PermGuildMembership must not move its player into a guild")

		after, found := k.GetPlayer(ctx, victim.Id)
		require.True(t, found)
		require.Empty(t, after.GuildId, "the player must not have been moved")
	})

	t.Run("granting the bit to that address is what unblocks it", func(t *testing.T) {
		ownerAcc := sdk.AccAddress("proxy_perm_primary02")
		joiner := testAppendPlayer(k, ctx, types.Player{
			Creator:        ownerAcc.String(),
			PrimaryAddress: ownerAcc.String(),
		})

		privKey := secp256k1.GenPrivKey()
		secondary := types.PubKeyToBech32(privKey.PubKey().Bytes())
		_ = k.SetPlayerIndexForAddress(ctx, secondary, joiner.Index)
		require.Equal(t, joiner.Index, k.GetPlayerIndexFromAddress(ctx, secondary))
		k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(secondary),
			types.PermPlay|types.PermGuildMembership)

		require.NoError(t, proxyJoin(privKey, secondary, 0))

		after, found := k.GetPlayer(ctx, joiner.Id)
		require.True(t, found)
		require.Equal(t, gs.Guild.Id, after.GuildId)
	})
}
