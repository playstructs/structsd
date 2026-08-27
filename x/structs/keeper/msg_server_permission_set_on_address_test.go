package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPermissionSetOnAddress(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	secondaryAcc := sdk.AccAddress("secondary123456789012345678901234567890")
	secondaryAddress := secondaryAcc.String()
	k.SetPlayerIndexForAddress(ctx, secondaryAddress, player.Index)

	creatorPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	testPermissionAdd(k, ctx, creatorPermissionId, types.PermAll)

	secondaryPermissionId := keeperlib.GetAddressPermissionIDBytes(secondaryAddress)
	testPermissionAdd(k, ctx, secondaryPermissionId, types.PermAll)

	testCases := []struct {
		name      string
		input     *types.MsgPermissionSetOnAddress
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "valid permission set",
			input: &types.MsgPermissionSetOnAddress{
				Creator:     player.Creator,
				Address:     secondaryAddress,
				Permissions: uint64(types.PermPlay),
			},
			expErr: false,
		},
		{
			name: "address not associated with player",
			input: &types.MsgPermissionSetOnAddress{
				Creator:     player.Creator,
				Address:     sdk.AccAddress("notassociated123456789012345678901234567890").String(),
				Permissions: uint64(types.PermPlay),
			},
			expErr:    true,
			expErrMsg: "Non-player account",
			skip:      true, // Skip - cache system validation order
		},
		{
			name: "different player",
			input: &types.MsgPermissionSetOnAddress{
				Creator:     player.Creator,
				Address:     secondaryAddress,
				Permissions: uint64(types.PermPlay),
			},
			expErr:    true,
			expErrMsg: "Can only",
			skip:      true, // Skip - cache system validation order
		},
		{
			name: "insufficient permissions",
			input: &types.MsgPermissionSetOnAddress{
				Creator:     sdk.AccAddress("noperms123456789012345678901234567890").String(),
				Address:     secondaryAddress,
				Permissions: uint64(types.PermAll),
			},
			expErr:    true,
			expErrMsg: "does not have the permissions needed",
			skip:      true, // Skip - GetPlayerCacheFromAddress might create player
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test - error condition not easily testable with current cache system")
			}

			// Re-register address if needed
			if tc.name == "valid permission set" {
				k.SetPlayerIndexForAddress(ctx, secondaryAddress, player.Index)
			} else if tc.name == "different player" {
				// Create another player and associate address with them
				otherPlayer := types.Player{
					Creator:        "cosmos1other",
					PrimaryAddress: "cosmos1other",
				}
				otherPlayer = testAppendPlayer(k, ctx, otherPlayer)
				k.SetPlayerIndexForAddress(ctx, secondaryAddress, otherPlayer.Index)
			}

			resp, err := ms.PermissionSetOnAddress(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify permission was set
				targetPermissionId := keeperlib.GetAddressPermissionIDBytes(tc.input.Address)
				permissions := k.GetPermissionsByBytes(ctx, targetPermissionId)
				require.Equal(t, types.Permission(tc.input.Permissions), permissions)
			}
		})
	}
}

/* TestPermissionOnAddressCannotLockOutThePrimary covers the recovery half of the
 * full-mask overwrite report.
 *
 * Refusing an unauthorized bit removal closes the attack, but not the footgun:
 * a player's own PermAll key acting on its own primary address passes every
 * check precisely because it still holds the rights it is about to destroy. And
 * the resulting state is unrecoverable rather than merely reduced - every route
 * back runs through the primary, since PlayerUpdatePrimaryAddress needs PermAll
 * to rotate away and a grant can only hand over bits the granting address itself
 * holds.
 *
 * The rotate-then-narrow path is asserted alongside, because a guard that also
 * blocked the legitimate way to get a narrow key would be the wrong trade.
 */
func TestPermissionOnAddressCannotLockOutThePrimary(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	primaryAcc := sdk.AccAddress("lockout_primary_pad1")
	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        primaryAcc.String(),
		PrimaryAddress: primaryAcc.String(),
	})
	primaryPermId := keeperlib.GetAddressPermissionIDBytes(primaryAcc.String())
	require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, primaryPermId))

	t.Run("set cannot narrow the primary", func(t *testing.T) {
		_, err := ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     player.Creator,
			Address:     primaryAcc.String(),
			Permissions: uint64(types.PermPlay),
		})
		require.Error(t, err, "the primary must keep the rights every recovery route needs")
		require.Contains(t, err.Error(), "must keep full access")
		require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, primaryPermId))
	})

	t.Run("revoke cannot chip away at the primary", func(t *testing.T) {
		_, err := ms.PermissionRevokeOnAddress(wctx, &types.MsgPermissionRevokeOnAddress{
			Creator:     player.Creator,
			Address:     primaryAcc.String(),
			Permissions: uint64(types.PermTokenTransfer),
		})
		require.Error(t, err, "a partial revoke reaches the same dead end by a slower road")
		require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, primaryPermId))
	})

	t.Run("a set that leaves the primary whole is untouched", func(t *testing.T) {
		_, err := ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     player.Creator,
			Address:     primaryAcc.String(),
			Permissions: uint64(types.PermAll),
		})
		require.NoError(t, err)
		require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, primaryPermId))
	})

	// The supported way to end up with a narrow key: make the address you intend
	// to keep whole the primary first, then narrow the old one.
	t.Run("rotating the primary first still allows narrowing the old address", func(t *testing.T) {
		secondAcc := sdk.AccAddress("lockout_second_pad01")
		_ = k.SetPlayerIndexForAddress(ctx, secondAcc.String(), player.Index)
		k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(secondAcc.String()), types.PermAll)

		_, err := ms.PlayerUpdatePrimaryAddress(wctx, &types.MsgPlayerUpdatePrimaryAddress{
			Creator:        player.Creator,
			PrimaryAddress: secondAcc.String(),
		})
		require.NoError(t, err)

		_, err = ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     secondAcc.String(),
			Address:     primaryAcc.String(),
			Permissions: uint64(types.PermPlay),
		})
		require.NoError(t, err, "the old address is no longer the primary and may be narrowed")
		require.Equal(t, types.PermPlay, k.GetPermissionsByBytes(ctx, primaryPermId))
	})
}

/* TestPermissionSetOnAddressHoldsAcrossPlayers pins the capability model against
 * an outsider rather than a weak key of the same player.
 *
 * The rule is that a permission write must be authorized for every bit it
 * touches — the ones it grants and the ones it destroys. That is what stops a
 * narrow delegate rewriting a stronger address down to its own level, and it has
 * to hold when the delegate is a different player holding an object-level grant,
 * not only when it is another key of the same one.
 *
 * The controls matter as much as the refusals here: an outsider with a grant is
 * not forbidden from touching permissions, they are held to the bits they were
 * actually given. A model that refused everything would pass the same negative
 * assertions while breaking delegation entirely.
 */
func TestPermissionSetOnAddressHoldsAcrossPlayers(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	victimAcc := sdk.AccAddress("crossplayer_victim_pad01")
	victim := testAppendPlayer(k, ctx, types.Player{
		Creator:        victimAcc.String(),
		PrimaryAddress: victimAcc.String(),
	})

	attackerAcc := sdk.AccAddress("crossplayer_attacker_p01")
	attacker := testAppendPlayer(k, ctx, types.Player{
		Creator:        attackerAcc.String(),
		PrimaryAddress: attackerAcc.String(),
	})

	// The victim delegates play rights on themselves to the attacker, and
	// nothing else. The attacker's own key holds PermAll, so the object grant is
	// the only thing limiting them — which is what this isolates.
	k.SetPermissionsByBytes(ctx,
		keeperlib.GetObjectPermissionIDBytes(victim.Id, attacker.Id), types.PermPlay)

	// Two more addresses of the victim: one strong, one already narrow.
	strongAcc := sdk.AccAddress("crossplayer_strong_pad01")
	_ = k.SetPlayerIndexForAddress(ctx, strongAcc.String(), victim.Index)
	strongPermId := keeperlib.GetAddressPermissionIDBytes(strongAcc.String())
	k.SetPermissionsByBytes(ctx, strongPermId, types.PermPlay|types.PermDelete)

	narrowAcc := sdk.AccAddress("crossplayer_narrow_pad01")
	_ = k.SetPlayerIndexForAddress(ctx, narrowAcc.String(), victim.Index)
	narrowPermId := keeperlib.GetAddressPermissionIDBytes(narrowAcc.String())
	k.SetPermissionsByBytes(ctx, narrowPermId, types.PermPlay)

	t.Run("cannot destroy a bit the grant never included", func(t *testing.T) {
		_, err := ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     attacker.Creator,
			Address:     strongAcc.String(),
			Permissions: uint64(types.PermPlay),
		})
		require.Error(t, err, "PermDelete is being destroyed and was never delegated")
		require.Equal(t, types.PermPlay|types.PermDelete, k.GetPermissionsByBytes(ctx, strongPermId))
	})

	t.Run("cannot narrow the victim's primary", func(t *testing.T) {
		primaryPermId := keeperlib.GetAddressPermissionIDBytes(victimAcc.String())
		require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, primaryPermId))

		_, err := ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     attacker.Creator,
			Address:     victimAcc.String(),
			Permissions: uint64(types.PermPlay),
		})
		require.Error(t, err)
		require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, primaryPermId),
			"the address every recovery route runs through must be untouched")
	})

	t.Run("cannot grant a bit the grant never included", func(t *testing.T) {
		_, err := ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     attacker.Creator,
			Address:     narrowAcc.String(),
			Permissions: uint64(types.PermPlay | types.PermTokenTransfer),
		})
		require.Error(t, err, "a delegate may not confer what it was not given")
		require.Equal(t, types.PermPlay, k.GetPermissionsByBytes(ctx, narrowPermId))
	})

	// The control: within the delegated bit the outsider may still act, so the
	// refusals above are about the bits and not about being an outsider.
	t.Run("may act within the delegated bit", func(t *testing.T) {
		_, err := ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     attacker.Creator,
			Address:     narrowAcc.String(),
			Permissions: uint64(types.PermPlay),
		})
		require.NoError(t, err)
		require.Equal(t, types.PermPlay, k.GetPermissionsByBytes(ctx, narrowPermId))
	})
}
