package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipInviteRevoke(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup)
	}{
		{
			name: "valid invite revoke",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_revoke_target")
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

				resp, err := ms.GuildMembershipInviteRevoke(wctx, &types.MsgGuildMembershipInviteRevoke{
					Creator:  gs.GuildOwner.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.GuildMembershipApplication)
				require.Equal(t, types.RegistrationStatus_revoked, resp.GuildMembershipApplication.RegistrationStatus)

				player, _ := k.GetPlayer(ctx, target.Id)
				require.NotEqual(t, gs.Guild.Id, player.GuildId)
			},
		},
		{
			name: "no pending invite",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_revoke_no_inv00")
				target := types.Player{
					Creator:        targetAcc.String(),
					PrimaryAddress: targetAcc.String(),
				}
				target = testAppendPlayer(k, ctx, target)

				_, err := ms.GuildMembershipInviteRevoke(wctx, &types.MsgGuildMembershipInviteRevoke{
					Creator:  gs.GuildOwner.Creator,
					GuildId:  "0-999",
					PlayerId: target.Id,
				})
				require.Error(t, err)
				require.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "non-member tries to revoke",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_revoke_targ01")
				target := types.Player{
					Creator:        targetAcc.String(),
					PrimaryAddress: targetAcc.String(),
				}
				target = testAppendPlayer(k, ctx, target)

				outsiderAcc := sdk.AccAddress("outsider_revoke_addr")
				outsider := types.Player{
					Creator:        outsiderAcc.String(),
					PrimaryAddress: outsiderAcc.String(),
				}
				outsider = testAppendPlayer(k, ctx, outsider)

				_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
					Creator:  gs.GuildOwner.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.NoError(t, err)

				_, err = ms.GuildMembershipInviteRevoke(wctx, &types.MsgGuildMembershipInviteRevoke{
					Creator:  outsider.Creator,
					GuildId:  gs.Guild.Id,
					PlayerId: target.Id,
				})
				require.Error(t, err)
				require.Contains(t, err.Error(), "not a member")
			},
		},
		{
			name: "unregistered creator",
			run: func(t *testing.T, k keeperlib.Keeper, ms types.MsgServer, ctx context.Context, wctx sdk.Context, gs testGuildSetup) {
				targetAcc := sdk.AccAddress("invite_revoke_targ02")
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

				unregAddr := sdk.AccAddress("unregistered_revoke0").String()
				_, err = ms.GuildMembershipInviteRevoke(wctx, &types.MsgGuildMembershipInviteRevoke{
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
