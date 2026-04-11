package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPlanetUpdateName(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("planetname123456789012345678901234")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	planet := testAppendPlanet(k, ctx, types.Planet{
		Creator: player.Creator,
		Owner:   player.Id,
	})

	planetPermId := keeperlib.GetObjectPermissionIDBytes(planet.Id, player.Id)
	testPermissionAdd(k, ctx, planetPermId, types.PermUpdate)

	testCases := []struct {
		name      string
		input     *types.MsgPlanetUpdateName
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid name",
			input: &types.MsgPlanetUpdateName{
				Creator:  player.Creator,
				PlanetId: planet.Id,
				Name:     "New Terra",
			},
			expErr: false,
		},
		{
			name: "valid long planet name",
			input: &types.MsgPlanetUpdateName{
				Creator:  player.Creator,
				PlanetId: planet.Id,
				Name:     "Kepler Four-Fifty-Two",
			},
			expErr: false,
		},
		{
			name: "name too short",
			input: &types.MsgPlanetUpdateName{
				Creator:  player.Creator,
				PlanetId: planet.Id,
				Name:     "ab",
			},
			expErr:    true,
			expErrMsg: "must be 3-25 characters",
		},
		{
			name: "name too long",
			input: &types.MsgPlanetUpdateName{
				Creator:  player.Creator,
				PlanetId: planet.Id,
				Name:     "abcdefghij1234567890abcdef",
			},
			expErr:    true,
			expErrMsg: "must be 3-25 characters",
		},
		{
			name: "object id pattern rejected",
			input: &types.MsgPlanetUpdateName{
				Creator:  player.Creator,
				PlanetId: planet.Id,
				Name:     "5-100",
			},
			expErr:    true,
			expErrMsg: "cannot resemble an object ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.PlanetUpdateName(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				updatedPlanet, found := k.GetPlanet(ctx, planet.Id)
				require.True(t, found)
				require.Equal(t, tc.input.Name, updatedPlanet.Name)
			}
		})
	}
}
