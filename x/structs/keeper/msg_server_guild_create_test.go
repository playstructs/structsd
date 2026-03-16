package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildCreate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	reactor := types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	}
	reactor = k.AppendReactor(ctx, reactor)

	reactorPermissionId := keeperlib.GetObjectPermissionIDBytes(reactor.Id, player.Id)
	testPermissionAdd(k, ctx, reactorPermissionId, types.PermAll)

	testCases := []struct {
		name      string
		input     *types.MsgGuildCreate
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "valid guild creation",
			input: &types.MsgGuildCreate{
				Creator:           player.Creator,
				ReactorId:         reactor.Id,
				Endpoint:          "test-endpoint",
				EntrySubstationId: "",
			},
			expErr: false,
		},
		{
			name: "missing reactor id",
			input: &types.MsgGuildCreate{
				Creator:           player.Creator,
				ReactorId:         "",
				Endpoint:          "test-endpoint",
				EntrySubstationId: "",
			},
			expErr:    true,
			expErrMsg: "reactor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test")
			}

			resp, err := ms.GuildCreate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.GuildId)

				guild, found := k.GetGuild(ctx, resp.GuildId)
				require.True(t, found)
				require.Equal(t, tc.input.Endpoint, guild.Endpoint)
			}
		})
	}
}
