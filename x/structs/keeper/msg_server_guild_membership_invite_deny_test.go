package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipInviteDeny(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup)
	}{
		{
			name: "valid invite deny",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_deny_target00")
				target := types.Player{
					Creator:        targetAcc.String(),
					PrimaryAddress: targetAcc.String(),
				}
				target = testAppendPlayer(k, ctx, target)

				_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
					Creator:  gs.GuildOwner.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.NoError(t, err)

				resp, err := ms.GuildMembershipInviteDeny(wctx, &types.MsgGuildMembershipInviteDeny{
					Creator:  target.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.GuildMembershipApplication)
				require.Equal(t, types.RegistrationStatus_denied, resp.GuildMembershipApplication.RegistrationStatus)

				player, _ := k.GetPlayer(ctx, target.Id)
				require.NotEqual(t, gs.Guild.Id, player.GuildId)
			},
		},
		{
			name: "no pending invite",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_deny_no_inv00")
				target := types.Player{
					Creator:        targetAcc.String(),
					PrimaryAddress: targetAcc.String(),
				}
				target = testAppendPlayer(k, ctx, target)

				_, err := ms.GuildMembershipInviteDeny(wctx, &types.MsgGuildMembershipInviteDeny{
					Creator:  target.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.Error(t, err)
				require.Contains(t, err.Error(), "not a member")
			},
		},
		{
			name: "join type mismatch",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_deny_mismatch0")
				target := types.Player{
					Creator:        targetAcc.String(),
					PrimaryAddress: targetAcc.String(),
				}
				target = testAppendPlayer(k, ctx, target)

				_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
					Creator:  target.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.NoError(t, err)

				_, err = ms.GuildMembershipInviteDeny(wctx, &types.MsgGuildMembershipInviteDeny{
					Creator:  target.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.Error(t, err)
				require.Contains(t, err.Error(), "join_type_mismatch")
			},
		},
		{
			name: "unregistered creator",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_deny_target01")
				target := types.Player{
					Creator:        targetAcc.String(),
					PrimaryAddress: targetAcc.String(),
				}
				target = testAppendPlayer(k, ctx, target)

				_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
					Creator:  gs.GuildOwner.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.NoError(t, err)

				unregAddr := sdk.AccAddress("unregistered_deny_addr").String()
				_, err = ms.GuildMembershipInviteDeny(wctx, &types.MsgGuildMembershipInviteDeny{
					Creator:  unregAddr,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.Error(t, err)
				require.Contains(t, err.Error(), "not associated")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, ms, ctx := setupMsgServer(t)
			wctx := sdk.UnwrapSDKContext(ctx)
			gs := testCreateGuild(k, ctx)
			tc.run(t, k, ms, ctx, wctx, gs)
		})
	}
}
