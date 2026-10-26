package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildUpdateEntrySubstationId(t *testing.T) {
	for _, name := range []string{"authorized third-party station", "guild-owned station", "missing guild", "missing station", "unregistered signer", "station permission absent", "key lacks guild bit", "key lacks connection bit", "empty destination"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			gs := testCreateGuild(k, ctx)
			other := testCreateGuild(k, ctx)
			station := other.Substation
			if name == "guild-owned station" {
				station.Owner = gs.Guild.Id
				k.SetSubstation(ctx, station)
			}
			stationPermission := keeperlib.GetObjectPermissionIDBytes(station.Id, gs.GuildOwner.Id)
			if name != "station permission absent" && name != "guild-owned station" {
				k.SetPermissionsByBytes(ctx, stationPermission, types.PermSubstationConnection)
			}
			keyPermission := keeperlib.GetAddressPermissionIDBytes(gs.GuildOwner.Creator)
			k.SetPermissionsByBytes(ctx, keyPermission, types.PermGuildSubstationUpdate|types.PermSubstationConnection)
			msg := &types.MsgGuildUpdateEntrySubstationId{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, EntrySubstationId: station.Id}
			wantError := ""
			switch name {
			case "missing guild":
				msg.GuildId, wantError = "0-999999", "not found"
			case "missing station":
				msg.EntrySubstationId, wantError = "4-999999", "not found"
			case "unregistered signer":
				msg.Creator, wantError = sdk.AccAddress("entry_station_unknown").String(), "player"
			case "station permission absent":
				wantError = "permission"
			case "key lacks guild bit":
				k.SetPermissionsByBytes(ctx, keyPermission, types.PermSubstationConnection)
				wantError = "permission"
			case "key lacks connection bit":
				k.SetPermissionsByBytes(ctx, keyPermission, types.PermGuildSubstationUpdate)
				wantError = "permission"
			case "empty destination":
				msg.EntrySubstationId = ""
			}
			beforeGrid := k.GetAllGridExport(ctx)
			beforePermissions := k.GetPermissionsByObject(ctx, station.Id)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			resp, err := ms.GuildUpdateEntrySubstationId(ctx, msg)
			require.NotNil(t, resp)
			actual, found := k.GetGuild(ctx, gs.Guild.Id)
			require.True(t, found)
			expected := gs.Guild
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
			} else {
				require.NoError(t, err)
				if msg.EntrySubstationId != "" {
					expected.EntrySubstationId = station.Id
				}
			}
			require.Equal(t, expected, actual)
			guildEvents := 0
			for _, event := range ctx.EventManager().ABCIEvents() {
				require.NotContains(t, []string{"structs.structs.EventPermission", "structs.structs.EventSubstation"}, event.Type)
				if event.Type == "structs.structs.EventGuild" {
					decoded, decodeErr := sdk.ParseTypedEvent(event)
					require.NoError(t, decodeErr)
					require.Equal(t, &actual, decoded.(*types.EventGuild).Guild)
					guildEvents++
				}
			}
			if wantError != "" || msg.EntrySubstationId == "" {
				require.Zero(t, guildEvents)
			} else {
				require.Equal(t, 1, guildEvents)
			}
			storedStation, found := k.GetSubstation(ctx, station.Id)
			require.True(t, found)
			require.Equal(t, station, storedStation)
			require.Equal(t, beforeGrid, k.GetAllGridExport(ctx))
			require.Equal(t, beforePermissions, k.GetPermissionsByObject(ctx, station.Id))
			player, found := k.GetPlayer(ctx, gs.GuildOwner.Id)
			require.True(t, found)
			require.Equal(t, gs.GuildOwner, player, "updating future admission does not migrate existing members")
		})
	}
}
