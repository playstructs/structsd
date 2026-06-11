package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgPlayerUpdatePfpClientRenderAttributes(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("playerupdatepfpcrattrs1234567890123")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	testCases := []struct {
		name      string
		input     *types.MsgPlayerUpdatePfpClientRenderAttributes
		expErr    bool
		expErrMsg string
		expStored string
	}{
		{
			name: "valid object",
			input: &types.MsgPlayerUpdatePfpClientRenderAttributes{
				Creator:                   player.Creator,
				PlayerId:                  player.Id,
				PfpClientRenderAttributes: `{"head":1234,"neck":1234,"body":1234,"arms":1234,"background":1234}`,
			},
			expErr:    false,
			expStored: `{"head":1234,"neck":1234,"body":1234,"arms":1234,"background":1234}`,
		},
		{
			name: "whitespace is compacted on store",
			input: &types.MsgPlayerUpdatePfpClientRenderAttributes{
				Creator:                   player.Creator,
				PlayerId:                  player.Id,
				PfpClientRenderAttributes: "{\n  \"head\": 1234,\n  \"schema\": 7\n}",
			},
			expErr:    false,
			expStored: `{"head":1234,"schema":7}`,
		},
		{
			name: "clear attributes",
			input: &types.MsgPlayerUpdatePfpClientRenderAttributes{
				Creator:                   player.Creator,
				PlayerId:                  player.Id,
				PfpClientRenderAttributes: "",
			},
			expErr:    false,
			expStored: "",
		},
		{
			name: "rejects non-object json",
			input: &types.MsgPlayerUpdatePfpClientRenderAttributes{
				Creator:                   player.Creator,
				PlayerId:                  player.Id,
				PfpClientRenderAttributes: `[1,2,3]`,
			},
			expErr:    true,
			expErrMsg: "valid JSON object",
		},
		{
			name: "rejects malformed json",
			input: &types.MsgPlayerUpdatePfpClientRenderAttributes{
				Creator:                   player.Creator,
				PlayerId:                  player.Id,
				PfpClientRenderAttributes: `{"head":}`,
			},
			expErr:    true,
			expErrMsg: "valid JSON object",
		},
		{
			name: "rejects oversized blob",
			input: &types.MsgPlayerUpdatePfpClientRenderAttributes{
				Creator:                   player.Creator,
				PlayerId:                  player.Id,
				PfpClientRenderAttributes: `{"x":"` + strings.Repeat("a", types.MaxPfpClientRenderAttributesBytes) + `"}`,
			},
			expErr:    true,
			expErrMsg: "at most",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.PlayerUpdatePfpClientRenderAttributes(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				updatedPlayer, found := k.GetPlayer(ctx, player.Id)
				require.True(t, found)
				require.Equal(t, tc.expStored, updatedPlayer.PfpClientRenderAttributes)
			}
		})
	}
}
