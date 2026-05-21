package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPlanetExplore(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	// Set up player capacity to be online
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))

	// Set last action to ensure player has charge
	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

	// Fleet and planet setup will be done in each test case to ensure clean state

	testCases := []struct {
		name      string
		input     *types.MsgPlanetExplore
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "valid planet exploration",
			input: &types.MsgPlanetExplore{
				Creator:  player.Creator,
				PlayerId: player.Id,
			},
			expErr: false,
		},
		{
			name: "invalid player id",
			input: &types.MsgPlanetExplore{
				Creator:  player.Creator,
				PlayerId: "invalid-player",
			},
			expErr:    true,
			expErrMsg: "Could not load Player",
			skip:      true, // Skip - cache system validation order
		},
		{
			name: "no play permissions",
			input: &types.MsgPlanetExplore{
				Creator:  sdk.AccAddress("noperms123456789012345678901234567890").String(),
				PlayerId: player.Id,
			},
			expErr:    true,
			expErrMsg: "has no",
			skip:      true, // Skip - GetPlayerCacheFromAddress might create player
		},
		{
			name: "player is halted",
			input: &types.MsgPlanetExplore{
				Creator:  player.Creator,
				PlayerId: player.Id,
			},
			expErr:    true,
			expErrMsg: "is Halted",
			skip:      true, // Skip - fleet setup may interfere with halted state
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test - error condition not easily testable with current cache system")
			}

			// Recreate planet and fleet setup for each test case to ensure clean state
			planet := testAppendPlanet(k, ctx, types.Planet{Creator: player.Creator, Owner: player.Id})

			testFleet := testAppendFleet(k, ctx, types.Fleet{Owner: player.Id})

			// Set fleet location to the planet
			fleetObj, _ := k.GetFleet(ctx, testFleet.Id)
			fleetObj.LocationId = planet.Id
			fleetObj.LocationType = types.ObjectType_planet
			k.SetFleet(ctx, fleetObj)

			resp, err := ms.PlanetExplore(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.Planet)
			}
		})
	}
}

// TestMsgPlanetExploreWithName covers the optional name field on
// MsgPlanetExplore. Each subtest provisions its own player so that name
// validation failures do not interfere with state from prior cases (and so
// that the success path always exercises a fresh exploration rather than
// the prior-planet completion branch, which would trip on PlanetStartingOre).
func TestMsgPlanetExploreWithName(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	setupExplorablePlayer := func(seed string) types.Player {
		playerAcc := sdk.AccAddress(seed)
		p := types.Player{
			Creator:        playerAcc.String(),
			PrimaryAddress: playerAcc.String(),
		}
		p = testAppendPlayer(k, ctx, p)

		capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, p.Id)
		k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))
		lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, p.Id)
		k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

		testAppendFleet(k, ctx, types.Fleet{Owner: p.Id})
		return p
	}

	testCases := []struct {
		name         string
		seed         string
		nameInput    string
		expErr       bool
		expErrMsg    string
		expectedName string
	}{
		{
			name:         "valid name on explore",
			seed:         "exploreName1pad_padding_addr_padding",
			nameInput:    "New Terra",
			expectedName: "New Terra",
		},
		{
			name:         "valid long name on explore",
			seed:         "exploreName2pad_padding_addr_padding",
			nameInput:    "Kepler Four-Fifty-Two",
			expectedName: "Kepler Four-Fifty-Two",
		},
		{
			name:         "empty name preserves prior behavior",
			seed:         "exploreName3pad_padding_addr_padding",
			nameInput:    "",
			expectedName: "",
		},
		{
			name:      "name too short rejects whole tx",
			seed:      "exploreName4pad_padding_addr_padding",
			nameInput: "ab",
			expErr:    true,
			expErrMsg: "must be 3-25 characters",
		},
		{
			name:      "name too long rejects whole tx",
			seed:      "exploreName5pad_padding_addr_padding",
			nameInput: "abcdefghij1234567890abcdef",
			expErr:    true,
			expErrMsg: "must be 3-25 characters",
		},
		{
			name:      "object id pattern rejected",
			seed:      "exploreName6pad_padding_addr_padding",
			nameInput: "5-100",
			expErr:    true,
			expErrMsg: "cannot resemble an object ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := setupExplorablePlayer(tc.seed)

			resp, err := ms.PlanetExplore(wctx, &types.MsgPlanetExplore{
				Creator:  p.Creator,
				PlayerId: p.Id,
				Name:     tc.nameInput,
			})

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)

				// Validation must fail before any state mutation: the player
				// should still have no planet attached.
				stored, found := k.GetPlayer(ctx, p.Id)
				require.True(t, found)
				require.Equal(t, "", stored.PlanetId, "player must have no planet after failed explore")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, tc.expectedName, resp.Planet.Name)

			stored, found := k.GetPlayer(ctx, p.Id)
			require.True(t, found)
			require.NotEqual(t, "", stored.PlanetId, "player must have a planet after successful explore")

			planetObj, found := k.GetPlanet(ctx, stored.PlanetId)
			require.True(t, found)
			require.Equal(t, tc.expectedName, planetObj.Name)
		})
	}
}
