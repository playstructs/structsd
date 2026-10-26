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
	for _, name := range []string{"owner with exact key bit", "explicit delegate", "missing substation", "wrong object type", "unregistered signer", "unprivileged player", "owner key lacks source", "delegate key lacks source"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			gs := testCreateGuild(k, ctx)
			address := sdk.AccAddress("create_other_player_")
			other := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
			msg := &types.MsgProviderCreate{
				Creator: gs.GuildOwner.Creator, SubstationId: gs.Substation.Id,
				Rate: sdk.NewInt64Coin("ualpha", 100), AccessPolicy: types.ProviderAccessPolicy_openMarket,
				CapacityMinimum: 100, CapacityMaximum: 1000, DurationMinimum: 1, DurationMaximum: 10,
				ProviderCancellationPenalty: math.LegacyNewDec(1), ConsumerCancellationPenalty: math.LegacyNewDec(1),
			}
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermSourceAllocation)
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(other.Creator), types.PermSourceAllocation)
			wantError, wantOwner := "", gs.GuildOwner.Id
			switch name {
			case "explicit delegate", "delegate key lacks source":
				msg.Creator, wantOwner = other.Creator, other.Id
				k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(gs.Substation.Id, other.Id), types.PermSourceAllocation)
				if name == "delegate key lacks source" {
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermAll &^ types.PermSourceAllocation)
					wantError = "permission"
				}
			case "missing substation":
				msg.SubstationId, wantError = "4-999999", "not found"
			case "wrong object type":
				msg.SubstationId, wantError = gs.GuildOwner.Id, "not found"
			case "unregistered signer":
				msg.Creator, wantError = sdk.AccAddress("create_unknown_key__").String(), "player"
			case "unprivileged player":
				msg.Creator, wantError = other.Creator, "permission"
			case "owner key lacks source":
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermAll &^ types.PermSourceAllocation)
				wantError = "permission"
			}
			permissionsBefore := k.GetAllPermissionExport(ctx)
			gridBefore := k.GetAllGridExport(ctx)
			countBefore := k.GetProviderCount(ctx)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			response, err := ms.ProviderCreate(ctx, msg)
			require.NotNil(t, response)
			providers := k.GetAllProvider(ctx)
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
				require.Empty(t, providers)
				require.Equal(t, countBefore, k.GetProviderCount(ctx))
				require.Equal(t, permissionsBefore, k.GetAllPermissionExport(ctx))
				require.Equal(t, gridBefore, k.GetAllGridExport(ctx))
			} else {
				require.NoError(t, err)
				require.Len(t, providers, 1)
				require.Equal(t, countBefore+1, k.GetProviderCount(ctx))
				provider := providers[0]
				require.Equal(t, wantOwner, provider.Owner)
				require.Equal(t, msg.Creator, provider.Creator)
				require.Equal(t, msg.SubstationId, provider.SubstationId)
				require.Equal(t, msg.Rate, provider.Rate)
				require.Equal(t, msg.AccessPolicy, provider.AccessPolicy)
				require.Equal(t, msg.CapacityMinimum, provider.CapacityMinimum)
				require.Equal(t, msg.CapacityMaximum, provider.CapacityMaximum)
				require.Equal(t, msg.DurationMinimum, provider.DurationMinimum)
				require.Equal(t, msg.DurationMaximum, provider.DurationMaximum)
				require.Equal(t, msg.ProviderCancellationPenalty, provider.ProviderCancellationPenalty)
				require.Equal(t, msg.ConsumerCancellationPenalty, provider.ConsumerCancellationPenalty)
				require.Equal(t, types.PermProviderAll, k.GetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(provider.Id, wantOwner)))
			}
			providerEvents := 0
			for _, event := range ctx.EventManager().ABCIEvents() {
				if event.Type == "structs.structs.EventProvider" {
					decoded, err := sdk.ParseTypedEvent(event)
					require.NoError(t, err)
					require.Len(t, providers, 1)
					require.Equal(t, &providers[0], decoded.(*types.EventProvider).Provider)
					providerEvents++
				}
			}
			if wantError == "" {
				require.Equal(t, 1, providerEvents)
			} else {
				require.Zero(t, providerEvents)
			}
			substation, found := k.GetSubstation(ctx, gs.Substation.Id)
			require.True(t, found)
			require.Equal(t, gs.Substation, substation)
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
