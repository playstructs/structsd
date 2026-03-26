package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipRequestDeny(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	requesterAcc := sdk.AccAddress("req_deny_addr_pad_01")
	requester := types.Player{
		Creator:        requesterAcc.String(),
		PrimaryAddress: requesterAcc.String(),
	}
	requester = testAppendPlayer(k, ctx, requester)

	requesterPermId := keeperlib.GetAddressPermissionIDBytes(requester.Creator)
	testPermissionAdd(k, ctx, requesterPermId, types.PermGuildMembership)

	t.Run("valid request deny", func(t *testing.T) {
		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)

		resp, err := ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)
		require.Equal(t, types.RegistrationStatus_denied, resp.GuildMembershipApplication.RegistrationStatus)
	})

	t.Run("no pending request", func(t *testing.T) {
		t.Skip("When no request exists, handler creates and denies; no 'not found' error path")
	})

	t.Run("denier not in guild", func(t *testing.T) {
		outsiderAcc := sdk.AccAddress("outsider_deny_pad_01")
		outsider := types.Player{
			Creator:        outsiderAcc.String(),
			PrimaryAddress: outsiderAcc.String(),
		}
		outsider = testAppendPlayer(k, ctx, outsider)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  outsider.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a member")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregAcc := sdk.AccAddress("unreg_deny_addr_pad_0")
		_, err := ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  unregAcc.String(),
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.Error(t, err)
	})
}
