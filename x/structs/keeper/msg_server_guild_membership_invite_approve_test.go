package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipInviteApprove(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup)
	}{
		{
			name: "valid invite approve",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_approve_addr0")
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

				resp, err := ms.GuildMembershipInviteApprove(wctx, &types.MsgGuildMembershipInviteApprove{
					Creator:  target.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.GuildMembershipApplication)

				player, _ := k.GetPlayer(ctx, target.Id)
				require.Equal(t, gs.Guild.Id, player.GuildId)
			},
		},
		{
			name: "cross-guild invite approve migrates player",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				gsB := testCreateGuild(k, ctx)

				playerAcc := sdk.AccAddress("invite_crossguild_a0")
				player := types.Player{
					Creator:        playerAcc.String(),
					PrimaryAddress: playerAcc.String(),
					GuildId:        gs.Guild.Id,
				}
				player = testAppendPlayer(k, ctx, player)

				_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
					Creator:  gsB.GuildOwner.Creator,
					GuildId:  gsB.Guild.Id,
					PlayerId: player.Id,
				})
				require.NoError(t, err)

				resp, err := ms.GuildMembershipInviteApprove(wctx, &types.MsgGuildMembershipInviteApprove{
					Creator:  player.Creator,
					GuildId:  gsB.Guild.Id,
					PlayerId: player.Id,
				})
				require.NoError(t, err)
				require.NotNil(t, resp)

				p, _ := k.GetPlayer(ctx, player.Id)
				require.Equal(t, gsB.Guild.Id, p.GuildId)
			},
		},
		{
			name: "no pending invite",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_no_pending_ad")
				target := types.Player{
					Creator:        targetAcc.String(),
					PrimaryAddress: targetAcc.String(),
				}
				target = testAppendPlayer(k, ctx, target)

				_, err := ms.GuildMembershipInviteApprove(wctx, &types.MsgGuildMembershipInviteApprove{
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
				targetAcc := sdk.AccAddress("invite_mismatch_addr0")
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

				_, err = ms.GuildMembershipInviteApprove(wctx, &types.MsgGuildMembershipInviteApprove{
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
				targetAcc := sdk.AccAddress("invite_unreg_target00")
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

				unregAddr := sdk.AccAddress("unregistered_addr000").String()
				_, err = ms.GuildMembershipInviteApprove(wctx, &types.MsgGuildMembershipInviteApprove{
					Creator:  unregAddr,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.Error(t, err)
				require.Contains(t, err.Error(), "not associated")
			},
		},
		{
			name: "invalid substation override",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_sub_override0")
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

				_, err = ms.GuildMembershipInviteApprove(wctx, &types.MsgGuildMembershipInviteApprove{
					Creator:      target.Creator,
					GuildId:      gs.Guild.Id,
					PlayerId:     target.Id,
					SubstationId: "4-999",
				})
				require.Error(t, err)
				require.Contains(t, err.Error(), "not found")
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
