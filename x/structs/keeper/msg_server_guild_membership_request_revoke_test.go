package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipRequestRevoke(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	requesterAcc := sdk.AccAddress("req_revoke_addr_pad0")
	requester := types.Player{
		Creator:        requesterAcc.String(),
		PrimaryAddress: requesterAcc.String(),
	}
	requester = testAppendPlayer(k, ctx, requester)

	requesterPermId := keeperlib.GetAddressPermissionIDBytes(requester.Creator)
	testPermissionAdd(k, ctx, requesterPermId, types.PermGuildMembership)

	t.Run("valid request revoke", func(t *testing.T) {
		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)

		resp, err := ms.GuildMembershipRequestRevoke(wctx, &types.MsgGuildMembershipRequestRevoke{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)
		require.Equal(t, types.RegistrationStatus_revoked, resp.GuildMembershipApplication.RegistrationStatus)
	})

	t.Run("no pending request", func(t *testing.T) {
		_, err := ms.GuildMembershipRequestRevoke(wctx, &types.MsgGuildMembershipRequestRevoke{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: "1-999",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregAcc := sdk.AccAddress("unreg_revoke_addr_pad0")
		_, err := ms.GuildMembershipRequestRevoke(wctx, &types.MsgGuildMembershipRequestRevoke{
			Creator:  unregAcc.String(),
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.Error(t, err)
	})
}
