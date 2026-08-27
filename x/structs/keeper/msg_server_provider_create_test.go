package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgProviderCreate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	// Create a substation
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     player.Id,
	}
	createdAllocation, err := testAppendAllocation(k, ctx, allocation, 100)
	require.NoError(t, err)

	substation, _, err := testAppendSubstation(k, ctx, createdAllocation, player)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		input     *types.MsgProviderCreate
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "valid provider creation",
			input: &types.MsgProviderCreate{
				Creator:                     player.Creator,
				SubstationId:                substation.Id,
				Rate:                        sdk.NewCoin("token", math.NewInt(100)),
				AccessPolicy:                types.ProviderAccessPolicy_openMarket,
				CapacityMinimum:             100,
				CapacityMaximum:             1000,
				DurationMinimum:             1,
				DurationMaximum:             10,
				ProviderCancellationPenalty: math.LegacyNewDec(1),
				ConsumerCancellationPenalty: math.LegacyNewDec(1),
			},
			expErr: false,
		},
		{
			name: "invalid substation",
			input: &types.MsgProviderCreate{
				Creator:                     player.Creator,
				SubstationId:                "invalid-substation",
				Rate:                        sdk.NewCoin("token", math.NewInt(100)),
				AccessPolicy:                types.ProviderAccessPolicy_openMarket,
				CapacityMinimum:             100,
				CapacityMaximum:             1000,
				DurationMinimum:             1,
				DurationMaximum:             10,
				ProviderCancellationPenalty: math.LegacyNewDec(1),
				ConsumerCancellationPenalty: math.LegacyNewDec(1),
			},
			expErr:    true,
			expErrMsg: "not found",
			skip:      true, // Skip - cache system validation order
		},
		{
			name: "no permissions on substation",
			input: &types.MsgProviderCreate{
				Creator:                     sdk.AccAddress("noperms123456789012345678901234567890").String(),
				SubstationId:                substation.Id,
				Rate:                        sdk.NewCoin("token", math.NewInt(100)),
				AccessPolicy:                types.ProviderAccessPolicy_openMarket,
				CapacityMinimum:             100,
				CapacityMaximum:             1000,
				DurationMinimum:             1,
				DurationMaximum:             10,
				ProviderCancellationPenalty: math.LegacyNewDec(1),
				ConsumerCancellationPenalty: math.LegacyNewDec(1),
			},
			expErr:    true,
			expErrMsg: "has no",
			skip:      true, // Skip - GetPlayerCacheFromAddress might create player
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test - error condition not easily testable with current cache system")
			}

			resp, err := ms.ProviderCreate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify provider was created by checking substation has providers
				providers := k.GetAllProvider(ctx)
				found := false
				for _, p := range providers {
					if p.SubstationId == substation.Id && p.Owner == player.Id {
						found = true
						break
					}
				}
				require.True(t, found, "Provider should be created")
			}
		})
	}
}

/* TestProviderCreateRequiresARealSubstation is the regression on a provider
 * backed by something that is not a substation.
 *
 * cc.GetSubstation is a cache allocator: it wraps any string and reads nothing.
 * The permission check that followed was not a backstop, because object
 * permissions are keyed by the raw id string and every player is granted PermAll
 * on their own player id at registration - so submitting that id passed
 * CanAllocateAsSourceBy against a substation that was never there.
 *
 * What made it more than an odd record is that grid attribute ids are derived
 * from the object id alone. A provider whose substation id is a player id reads
 * its available capacity from that player's own capacity and load counters, so
 * it sells grid the substation accounting knows nothing about.
 */
func TestProviderCreateRequiresARealSubstation(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	acc := sdk.AccAddress("providersub_player_pad_addr_00001")
	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        acc.String(),
		PrimaryAddress: acc.String(),
	})

	// The player's own capacity, which is what a provider keyed to the player id
	// would have ended up selling.
	k.SetGridAttribute(ctx,
		keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), 100000)

	baseProvider := func(substationId string) *types.MsgProviderCreate {
		return &types.MsgProviderCreate{
			Creator:                     player.Creator,
			SubstationId:                substationId,
			Rate:                        sdk.NewCoin("ualpha", math.NewInt(100)),
			AccessPolicy:                types.ProviderAccessPolicy_openMarket,
			CapacityMinimum:             100,
			CapacityMaximum:             1000,
			DurationMinimum:             1,
			DurationMaximum:             10,
			ProviderCancellationPenalty: math.LegacyNewDec(1),
			ConsumerCancellationPenalty: math.LegacyNewDec(1),
		}
	}

	t.Run("a player id is not a substation", func(t *testing.T) {
		// CurrentContext.NewPlayer grants a player PermAll on their own object
		// at registration; the test helper takes a shortcut and only sets the
		// address permission, so put the attacker in the position a real
		// registration would have.
		k.SetPermissionsByBytes(ctx,
			keeperlib.GetObjectPermissionIDBytes(player.Id, player.Id), types.PermAll)
		require.Equal(t, types.PermAll,
			k.GetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(player.Id, player.Id)))

		_, err := ms.ProviderCreate(wctx, baseProvider(player.Id))
		require.Error(t, err, "holding rights on an object does not make it a substation")
	})

	t.Run("a substation id that names nothing is refused", func(t *testing.T) {
		_, err := ms.ProviderCreate(wctx, baseProvider(
			keeperlib.GetObjectID(types.ObjectType_substation, 9999)))
		require.Error(t, err, "the right namespace is not the same as an existing record")
	})

	t.Run("a malformed id is refused", func(t *testing.T) {
		_, err := ms.ProviderCreate(wctx, baseProvider("not-an-object-id"))
		require.Error(t, err)
	})

	// The control: a real substation the player may allocate from still works,
	// so the refusals above are about the id and not about the handler.
	t.Run("a real substation still creates a provider", func(t *testing.T) {
		sourceObjectId := "providersub-source"
		k.SetGridAttribute(ctx,
			keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId), 1000)

		allocation, allocErr := testAppendAllocation(k, ctx, types.Allocation{
			SourceObjectId: sourceObjectId,
			Type:           types.AllocationType_static,
			Controller:     player.Id,
		}, 100)
		require.NoError(t, allocErr)

		substation, _, substationErr := testAppendSubstation(k, ctx, allocation, player)
		require.NoError(t, substationErr)

		countBefore := k.GetProviderCount(ctx)

		resp, err := ms.ProviderCreate(wctx, baseProvider(substation.Id))
		require.NoError(t, err)
		require.NotNil(t, resp)

		require.Equal(t, countBefore+1, k.GetProviderCount(ctx),
			"a valid substation must actually produce a provider")

		created, found := k.GetProvider(ctx, keeperlib.GetObjectID(types.ObjectType_provider, countBefore))
		require.True(t, found)
		require.Equal(t, substation.Id, created.SubstationId)
	})
}
