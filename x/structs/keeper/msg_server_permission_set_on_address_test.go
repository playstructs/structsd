package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPermissionSetOnAddress(t *testing.T) {
	for _, name := range []string{"primary signer", "exact secondary signer", "unregistered signer", "unregistered target", "other player target", "restricted secondary signer", "primary cannot be narrowed", "key lacks erased bits"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			primary := sdk.AccAddress("permission_primary01")
			player := testAppendPlayer(k, ctx, types.Player{Creator: primary.String(), PrimaryAddress: primary.String()})
			target := sdk.AccAddress("permission_subject01")
			signer := sdk.AccAddress("permission_signer001")
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, target.String(), player.Index))
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, signer.String(), player.Index))
			foreign := sdk.AccAddress("permission_foreign01")
			testAppendPlayer(k, ctx, types.Player{Creator: foreign.String(), PrimaryAddress: foreign.String()})
			targetID := keeperlib.GetAddressPermissionIDBytes(target.String())
			k.SetPermissionsByBytes(ctx, targetID, types.PermUpdate|types.PermDelete)
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(signer.String()), types.PermPlay|types.PermUpdate|types.PermDelete)
			creator, subject, permissions := primary.String(), target.String(), types.PermPlay
			rejected := false
			switch name {
			case "exact secondary signer":
				creator = signer.String()
			case "unregistered signer":
				creator, rejected = sdk.AccAddress("permission_unknown01").String(), true
			case "unregistered target":
				subject, rejected = sdk.AccAddress("permission_unknown01").String(), true
			case "other player target":
				subject, rejected = foreign.String(), true
			case "restricted secondary signer":
				creator, rejected = signer.String(), true
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.Permissionless)
			case "primary cannot be narrowed":
				subject, rejected = primary.String(), true
			case "key lacks erased bits":
				creator, rejected = signer.String(), true
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermPlay)

			}
			before := k.GetAllPermissionExport(ctx)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			response, err := ms.PermissionSetOnAddress(ctx, &types.MsgPermissionSetOnAddress{Creator: creator, Address: subject, Permissions: uint64(permissions)})
			require.NotNil(t, response)
			permissionEvents := 0
			for _, event := range ctx.EventManager().ABCIEvents() {
				if event.Type == "structs.structs.EventPermission" {
					decoded, decodeErr := sdk.ParseTypedEvent(event)
					require.NoError(t, decodeErr)
					require.Equal(t, &types.PermissionRecord{PermissionId: string(targetID), Value: uint64(types.PermPlay)}, decoded.(*types.EventPermission).PermissionRecord)
					permissionEvents++
				}
			}
			if rejected {
				require.Error(t, err)
				if name == "primary cannot be narrowed" {
					require.ErrorContains(t, err, "must keep full access")
				}
				require.Equal(t, before, k.GetAllPermissionExport(ctx))
				require.Zero(t, permissionEvents)
			} else {
				require.NoError(t, err)
				require.Equal(t, types.PermPlay, k.GetPermissionsByBytes(ctx, targetID))
				require.Equal(t, 1, permissionEvents)
			}
			require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(primary.String())))
			require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(foreign.String())))
			stored, found := k.GetPlayer(ctx, player.Id)
			require.True(t, found)
			require.Equal(t, player, stored)
		})
	}
}

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
