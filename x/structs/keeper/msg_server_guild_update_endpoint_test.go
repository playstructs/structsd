package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
	"testing"
)

func TestMsgGuildUpdateEndpoint(t *testing.T) {
	for _, name := range []string{"owner with exact key bit", "explicit delegate", "missing guild", "unregistered signer", "unprivileged player", "key lacks specific bit"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			gs := testCreateGuild(k, ctx)
			address := sdk.AccAddress("guild_setting_actor01")
			other := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
			creator, guildID, wantError := gs.GuildOwner.Creator, gs.Guild.Id, ""
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermGuildEndpointUpdate)
			switch name {
			case "explicit delegate":
				creator = other.Creator
				k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, other.Id), types.PermGuildEndpointUpdate)
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermGuildEndpointUpdate)
			case "missing guild":
				guildID, wantError = "0-999999", "not found"
			case "unregistered signer":
				creator, wantError = sdk.AccAddress("guild_setting_unknown").String(), "player"
			case "unprivileged player":
				creator, wantError = other.Creator, "permission"
			case "key lacks specific bit":
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermAll&^types.PermGuildEndpointUpdate)
				wantError = "permission"
			}
			beforePermissions := k.GetPermissionsByObject(ctx, gs.Guild.Id)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			response, err := ms.GuildUpdateEndpoint(ctx, &types.MsgGuildUpdateEndpoint{Creator: creator, GuildId: guildID, Endpoint: "new-endpoint"})
			require.NotNil(t, response)
			actual, found := k.GetGuild(ctx, gs.Guild.Id)
			require.True(t, found)
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
				require.Equal(t, gs.Guild, actual)
				for _, event := range ctx.EventManager().ABCIEvents() {
					require.NotContains(t, []string{"structs.structs.EventGuild", "structs.structs.EventPermission"}, event.Type)
				}
			} else {
				require.NoError(t, err)
				expected := gs.Guild
				expected.Endpoint = "new-endpoint"
				require.Equal(t, expected, actual)
				guildEvents := 0
				for _, event := range ctx.EventManager().ABCIEvents() {
					if event.Type == "structs.structs.EventGuild" {
						decoded, decodeErr := sdk.ParseTypedEvent(event)
						require.NoError(t, decodeErr)
						require.Equal(t, &actual, decoded.(*types.EventGuild).Guild)
						guildEvents++
					}
				}
				require.Equal(t, 1, guildEvents)
			}
			require.Equal(t, beforePermissions, k.GetPermissionsByObject(ctx, gs.Guild.Id))
			actor, found := k.GetPlayer(ctx, other.Id)
			require.True(t, found)
			require.Equal(t, other, actor, "settings do not implicitly enroll or promote a delegate")
		})
	}
}
