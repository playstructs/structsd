package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// ThrottleTargetAuthorized is what stops an attacker reserving an object-global
// throttle key for somebody else's struct, fleet or player. The ante writes that
// reservation before the handler runs and the SDK commits it even when the
// handler rejects the message, so this answer is the only thing standing between
// a free 40-message transaction and 40 censored victim objects for the block.
//
// The cases below mirror the three ways PermissionCheck can say yes — owner,
// object-level delegation, guild rank — because an owner-equality shortcut would
// silently stop throttling every delegated action.

func testThrottleAuthPlayer(t *testing.T, k keeperlib.Keeper, ctx sdk.Context, name string) types.Player {
	t.Helper()

	acc := sdk.AccAddress(fmt.Sprintf("%s_padding_address_xx", name))
	return testAppendPlayer(k, ctx, types.Player{
		Creator:        acc.String(),
		PrimaryAddress: acc.String(),
	})
}

func TestThrottleTargetAuthorized_StructOwnerAllowed(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	owner := testThrottleAuthPlayer(t, k, sdkCtx, "structowner")
	structure := testAppendStruct(k, sdkCtx, types.Struct{Owner: owner.Id})

	require.True(t, k.ThrottleTargetAuthorized(ctx, owner.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashBuild))
	require.True(t, k.ThrottleTargetAuthorized(ctx, owner.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashMine))
	require.True(t, k.ThrottleTargetAuthorized(ctx, owner.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashRefine))
}

// The exploit, reduced: the attacker is a perfectly ordinary registered player
// whose own primary address holds PermAll, which is exactly what let the message
// past the ante's address-level check while naming a victim's struct.
func TestThrottleTargetAuthorized_StructStrangerDenied(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	victim := testThrottleAuthPlayer(t, k, sdkCtx, "structvictim")
	attacker := testThrottleAuthPlayer(t, k, sdkCtx, "structattack")
	structure := testAppendStruct(k, sdkCtx, types.Struct{Owner: victim.Id})

	attackerAddrPerm := k.GetPermissionsByBytes(sdkCtx, keeperlib.GetAddressPermissionIDBytes(attacker.PrimaryAddress))
	require.Equal(t, types.PermAll, attackerAddrPerm, "the attacker must hold PermAll on its own address, or this proves nothing")

	require.False(t, k.ThrottleTargetAuthorized(ctx, attacker.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashBuild))
	require.False(t, k.ThrottleTargetAuthorized(ctx, attacker.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashMine))
}

// PermissionCheck resolves object-level delegation against the owner *player*,
// not the struct, so the grant has to be recorded there for the helper to agree.
func TestThrottleTargetAuthorized_ObjectDelegateAllowed(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	owner := testThrottleAuthPlayer(t, k, sdkCtx, "delegowner")
	delegate := testThrottleAuthPlayer(t, k, sdkCtx, "delegplayer")
	structure := testAppendStruct(k, sdkCtx, types.Struct{Owner: owner.Id})

	require.False(t, k.ThrottleTargetAuthorized(ctx, delegate.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashMine))

	testPermissionAdd(k, sdkCtx, keeperlib.GetObjectPermissionIDBytes(owner.Id, delegate.Id), types.PermHashMine)

	require.True(t, k.ThrottleTargetAuthorized(ctx, delegate.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashMine))
	require.False(t, k.ThrottleTargetAuthorized(ctx, delegate.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashBuild),
		"a grant of one hash bit must not carry the others")
}

func TestThrottleTargetAuthorized_GuildRankAllowed(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	setup := testCreateGuild(k, sdkCtx)

	owner := testThrottleAuthPlayer(t, k, sdkCtx, "guildtarget")
	structure := testAppendStruct(k, sdkCtx, types.Struct{Owner: owner.Id})

	member := testThrottleAuthPlayer(t, k, sdkCtx, "guildmember")
	member.GuildId = setup.Guild.Id
	member.GuildRank = 2
	k.SetPlayer(sdkCtx, member)

	require.False(t, k.ThrottleTargetAuthorized(ctx, member.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashRefine))

	k.SetGuildRankPermissionStoreOnly(sdkCtx, owner.Id, setup.Guild.Id, types.PermHashRefine, 3)

	require.True(t, k.ThrottleTargetAuthorized(ctx, member.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashRefine))
}

func TestThrottleTargetAuthorized_FleetOwnerAllowedStrangerDenied(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	owner := testThrottleAuthPlayer(t, k, sdkCtx, "fleetowner")
	attacker := testThrottleAuthPlayer(t, k, sdkCtx, "fleetattack")
	fleet := testAppendFleet(k, sdkCtx, types.Fleet{Owner: owner.Id})

	require.True(t, k.ThrottleTargetAuthorized(ctx, owner.PrimaryAddress, types.ObjectType_fleet, fleet.Id, types.PermPlay))
	require.True(t, k.ThrottleTargetAuthorized(ctx, owner.PrimaryAddress, types.ObjectType_fleet, fleet.Id, types.PermHashRaid))

	require.False(t, k.ThrottleTargetAuthorized(ctx, attacker.PrimaryAddress, types.ObjectType_fleet, fleet.Id, types.PermPlay))
	require.False(t, k.ThrottleTargetAuthorized(ctx, attacker.PrimaryAddress, types.ObjectType_fleet, fleet.Id, types.PermHashRaid))
}

// The explore and register throttles key off a player id rather than an object,
// so the target resolves to itself and the owner shortcut has to fire.
func TestThrottleTargetAuthorized_PlayerSelfAllowedStrangerDenied(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	player := testThrottleAuthPlayer(t, k, sdkCtx, "exploreself")
	attacker := testThrottleAuthPlayer(t, k, sdkCtx, "exploreother")

	require.True(t, k.ThrottleTargetAuthorized(ctx, player.PrimaryAddress, types.ObjectType_player, player.Id, types.PermPlay))
	require.False(t, k.ThrottleTargetAuthorized(ctx, attacker.PrimaryAddress, types.ObjectType_player, player.Id, types.PermPlay))
}

// The signing key's own bits are a hard ceiling, so a secondary address that was
// never granted the hash bit cannot reserve on its player's own struct.
func TestThrottleTargetAuthorized_SigningAddressBitsAreACeiling(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	owner := testThrottleAuthPlayer(t, k, sdkCtx, "ceilingowner")
	structure := testAppendStruct(k, sdkCtx, types.Struct{Owner: owner.Id})

	weakAddr := sdk.AccAddress("ceilingweak_padding_addr").String()
	k.SetPlayerIndexForAddress(sdkCtx, weakAddr, owner.Index)
	k.SetPermissionsByBytes(sdkCtx, keeperlib.GetAddressPermissionIDBytes(weakAddr), types.PermPlay)

	require.True(t, k.ThrottleTargetAuthorized(ctx, weakAddr, types.ObjectType_player, owner.Id, types.PermPlay))
	require.False(t, k.ThrottleTargetAuthorized(ctx, weakAddr, types.ObjectType_struct, structure.Id, types.PermHashBuild))
}

func TestThrottleTargetAuthorized_UnregisteredAddressDenied(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	owner := testThrottleAuthPlayer(t, k, sdkCtx, "unregowner")
	structure := testAppendStruct(k, sdkCtx, types.Struct{Owner: owner.Id})

	stranger := sdk.AccAddress("unregistered_padding_add").String()

	require.False(t, k.ThrottleTargetAuthorized(ctx, stranger, types.ObjectType_struct, structure.Id, types.PermHashBuild))
}

// Every id here is attacker-supplied and reaches the helper without any prior
// validation, so the only acceptable answers are false and no panic.
func TestThrottleTargetAuthorized_MalformedTargetsAreDenied(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	player := testThrottleAuthPlayer(t, k, sdkCtx, "malformedcall")

	cases := []struct {
		name     string
		creator  string
		kind     types.ObjectType
		targetId string
	}{
		{"empty creator", "", types.ObjectType_player, player.Id},
		{"empty target", player.PrimaryAddress, types.ObjectType_player, ""},
		{"struct that does not exist", player.PrimaryAddress, types.ObjectType_struct, "5-9999"},
		{"player that does not exist", player.PrimaryAddress, types.ObjectType_player, "1-9999"},
		{"fleet that does not exist", player.PrimaryAddress, types.ObjectType_fleet, "9-9999"},
		{"fleet id with no prefix", player.PrimaryAddress, types.ObjectType_fleet, "garbage"},
		{"fleet id with a non-numeric index", player.PrimaryAddress, types.ObjectType_fleet, "9-abc"},
		{"fleet id with too many parts", player.PrimaryAddress, types.ObjectType_fleet, "9-1-1"},
		{"struct id shaped like a player", player.PrimaryAddress, types.ObjectType_struct, player.Id},
		{"kind no throttle uses", player.PrimaryAddress, types.ObjectType_planet, "2-1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				require.False(t, k.ThrottleTargetAuthorized(ctx, tc.creator, tc.kind, tc.targetId, types.PermPlay))
			})
		})
	}
}

// The helper runs inside the ante, whose writes the SDK commits even when the
// message fails. Anything it persisted would persist unconditionally, so it must
// persist nothing — it loads caches and never commits them.
func TestThrottleTargetAuthorized_WritesNothing(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	owner := testThrottleAuthPlayer(t, k, sdkCtx, "readonlyowner")
	structure := testAppendStruct(k, sdkCtx, types.Struct{Owner: owner.Id})

	playerCountBefore := k.GetPlayerCount(sdkCtx)
	structCountBefore := k.GetStructCount(sdkCtx)
	ownerBefore, _ := k.GetPlayer(sdkCtx, owner.Id)

	k.ThrottleTargetAuthorized(ctx, owner.PrimaryAddress, types.ObjectType_struct, structure.Id, types.PermHashBuild)
	k.ThrottleTargetAuthorized(ctx, owner.PrimaryAddress, types.ObjectType_player, "1-9999", types.PermPlay)
	k.ThrottleTargetAuthorized(ctx, owner.PrimaryAddress, types.ObjectType_fleet, "9-9999", types.PermHashRaid)

	require.Equal(t, playerCountBefore, k.GetPlayerCount(sdkCtx))
	require.Equal(t, structCountBefore, k.GetStructCount(sdkCtx))

	ownerAfter, found := k.GetPlayer(sdkCtx, owner.Id)
	require.True(t, found)
	require.Equal(t, ownerBefore, ownerAfter)

	_, phantomPlayer := k.GetPlayer(sdkCtx, "1-9999")
	require.False(t, phantomPlayer, "a target lookup must not conjure the object it failed to find")
}
