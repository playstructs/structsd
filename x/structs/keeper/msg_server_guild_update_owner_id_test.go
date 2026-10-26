package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildUpdateOwnerId(t *testing.T) {
	for _, name := range []string{"nonmember recipient", "same owner", "missing guild", "missing recipient", "unregistered signer", "unprivileged player", "key lacks admin"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			gs := testCreateGuild(k, ctx)
			address := sdk.AccAddress("guild_recipient_0001")
			recipient := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
			require.Empty(t, recipient.GuildId)
			oldID := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, gs.GuildOwner.Id)
			newID := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, recipient.Id)
			k.SetPermissionsByBytes(ctx, oldID, types.PermGuildAll|types.PermPlay)
			k.SetPermissionsByBytes(ctx, newID, types.PermPlay)
			beforePermissions := k.GetPermissionsByObject(ctx, gs.Guild.Id)
			msg := &types.MsgGuildUpdateOwnerId{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, Owner: recipient.Id}
			wantError := ""
			switch name {
			case "same owner":
				msg.Owner = gs.GuildOwner.Id
			case "missing guild":
				msg.GuildId = "0-999999"
				wantError = "not found"
			case "missing recipient":
				msg.Owner = "1-999999"
				wantError = "not found"
			case "unregistered signer":
				msg.Creator = sdk.AccAddress("guild_unknown_00001").String()
				wantError = "player"
			case "unprivileged player":
				msg.Creator = recipient.Creator
				wantError = "permission"
			case "key lacks admin":
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(gs.GuildOwner.Creator), types.PermUpdate)
				wantError = "permission"
			}
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			resp, err := ms.GuildUpdateOwnerId(ctx, msg)
			require.NotNil(t, resp)
			stored, found := k.GetGuild(ctx, gs.Guild.Id)
			require.True(t, found)
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
			}
			if wantError != "" || name == "same owner" {
				if wantError == "" {
					require.NoError(t, err)
				}
				require.Equal(t, gs.Guild, stored)
				require.Equal(t, beforePermissions, k.GetPermissionsByObject(ctx, gs.Guild.Id))
				for _, event := range ctx.EventManager().ABCIEvents() {
					require.NotContains(t, []string{"structs.structs.EventGuild", "structs.structs.EventPermission"}, event.Type)
				}
			} else {
				require.NoError(t, err)
				expected := gs.Guild
				expected.Owner = recipient.Id
				require.Equal(t, expected, stored)
				require.Equal(t, types.Permissionless, k.GetPermissionsByBytes(ctx, oldID))
				require.Equal(t, types.PermGuildAll|types.PermPlay, k.GetPermissionsByBytes(ctx, newID))
				guildEvents := 0
				for _, event := range ctx.EventManager().ABCIEvents() {
					if event.Type == "structs.structs.EventGuild" {
						decoded, decodeErr := sdk.ParseTypedEvent(event)
						require.NoError(t, decodeErr)
						require.Equal(t, &stored, decoded.(*types.EventGuild).Guild)
						guildEvents++
					}
				}
				require.Equal(t, 1, guildEvents)
			}
			// Ownership does not silently change either player's membership/rank.
			oldPlayer, found := k.GetPlayer(ctx, gs.GuildOwner.Id)
			require.True(t, found)
			require.Equal(t, gs.GuildOwner, oldPlayer)
			newPlayer, found := k.GetPlayer(ctx, recipient.Id)
			require.True(t, found)
			require.Equal(t, recipient, newPlayer)
		})
	}
}
