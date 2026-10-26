package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgProviderUpdateAccessPolicy(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		ownership := "player owned"
		if guildOwned {
			ownership = "guild owned"
		}
		t.Run(ownership, func(t *testing.T) {
			for _, name := range []string{"owner with exact key bit", "explicit delegate", "open market", "closed market", "missing provider", "unregistered signer", "unprivileged player", "key lacks update", "undeclared policy", "negative policy"} {
				t.Run(name, func(t *testing.T) {
					k, ms, goCtx := setupMsgServer(t)
					ctx := sdk.UnwrapSDKContext(goCtx)
					gs := testCreateGuild(k, ctx)
					address := sdk.AccAddress("provider_policy_actor")
					other := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
					ownerID := gs.GuildOwner.Id
					if guildOwned {
						ownerID = gs.Guild.Id
					}
					provider := testAppendProvider(k, ctx, types.Provider{
						Owner: ownerID, Creator: gs.GuildOwner.Creator, SubstationId: gs.Guild.EntrySubstationId,
						Rate: sdk.NewCoin("token", math.NewInt(100)), AccessPolicy: types.ProviderAccessPolicy_closedMarket,
						CapacityMinimum: 100, CapacityMaximum: 1000, DurationMinimum: 1, DurationMaximum: 10,
						ProviderCancellationPenalty: math.LegacyNewDec(1), ConsumerCancellationPenalty: math.LegacyNewDec(1),
					})
					creator, providerID, wantError := gs.GuildOwner.Creator, provider.Id, ""
					policy := types.ProviderAccessPolicy_guildMarket
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermUpdate)
					switch name {
					case "explicit delegate":
						creator = other.Creator
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(provider.Id, other.Id), types.PermUpdate)
						k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermUpdate)
					case "open market":
						policy = types.ProviderAccessPolicy_openMarket
					case "closed market":
						policy = types.ProviderAccessPolicy_closedMarket
					case "missing provider":
						providerID, wantError = "11-999999", "not found"
					case "unregistered signer":
						creator, wantError = sdk.AccAddress("provider_unknown_key").String(), "player"
					case "unprivileged player":
						creator, wantError = other.Creator, "permission"
					case "key lacks update":
						k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermAll&^types.PermUpdate)
						wantError = "permission"
					case "undeclared policy":
						policy, wantError = types.ProviderAccessPolicy(500), "invalid_policy"
					case "negative policy":
						policy, wantError = types.ProviderAccessPolicy(-1), "invalid_policy"
					}
					permissions := k.GetAllPermissionExport(ctx)
					grid := k.GetAllGridExport(ctx)
					ctx = ctx.WithEventManager(sdk.NewEventManager())
					response, err := ms.ProviderUpdateAccessPolicy(ctx, &types.MsgProviderUpdateAccessPolicy{Creator: creator, ProviderId: providerID, AccessPolicy: policy})
					require.NotNil(t, response)
					actual, found := k.GetProvider(ctx, provider.Id)
					require.True(t, found)
					expected := provider
					if wantError != "" {
						require.ErrorContains(t, err, wantError)
					} else {
						require.NoError(t, err)
						expected.AccessPolicy = policy
					}
					require.Equal(t, expected, actual)
					providerEvents := 0
					for _, event := range ctx.EventManager().ABCIEvents() {
						require.NotEqual(t, "structs.structs.EventPermission", event.Type)
						if event.Type == "structs.structs.EventProvider" {
							decoded, decodeErr := sdk.ParseTypedEvent(event)
							require.NoError(t, decodeErr)
							require.Equal(t, &expected, decoded.(*types.EventProvider).Provider)
							providerEvents++
						}
					}
					if wantError != "" {
						require.Zero(t, providerEvents)
					} else {
						require.Equal(t, 1, providerEvents)
					}
					require.Equal(t, permissions, k.GetAllPermissionExport(ctx))
					require.Equal(t, grid, k.GetAllGridExport(ctx))
					guild, found := k.GetGuild(ctx, gs.Guild.Id)
					require.True(t, found)
					require.Equal(t, gs.Guild, guild)
					actor, found := k.GetPlayer(ctx, other.Id)
					require.True(t, found)
					require.Equal(t, other, actor)
				})
			}
		})
	}
}

/* TestProviderHandlersRequireARealProvider is the regression on a phantom
 * provider clearing the permission gate.
 *
 * cc.GetProvider is a cache allocator: it wraps any string and reads nothing.
 * The permission check that followed was not a backstop, because object
 * permissions are keyed by the raw id string with no type namespacing and
 * registration grants every player PermAll on their own player id - so
 * submitting that id collided with a record that genuinely exists and
 * CanBeUpdatedBy passed on a provider that did not.
 *
 * What followed was a zero-valued provider being mutated and committed. That
 * commit panics in the KV store on the empty Provider.Id, and BaseApp turns the
 * panic into a failed transaction - which reads like a refusal and is not one.
 * The same shape wrote real state in AllocationTransfer, where the write was
 * keyed by something non-empty and nothing tripped.
 *
 * Every provider handler took the id the same way, so all of them are covered
 * here rather than only the one the report named.
 */
func TestProviderHandlersRequireARealProvider(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	acc := sdk.AccAddress("phantomprovider_player_pad_00001")
	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        acc.String(),
		PrimaryAddress: acc.String(),
	})

	// CurrentContext.NewPlayer grants this at registration; the test helper only
	// sets the address permission, so put the attacker where a real
	// registration would have.
	k.SetPermissionsByBytes(ctx,
		keeperlib.GetObjectPermissionIDBytes(player.Id, player.Id), types.PermAll)

	// The attacker's own player id, standing in for a provider id.
	phantom := player.Id

	providersBefore := k.GetProviderCount(ctx)

	t.Run("update access policy", func(t *testing.T) {
		_, err := ms.ProviderUpdateAccessPolicy(wctx, &types.MsgProviderUpdateAccessPolicy{
			Creator:      player.Creator,
			ProviderId:   phantom,
			AccessPolicy: types.ProviderAccessPolicy_openMarket,
		})
		require.Error(t, err, "holding rights on an object does not make it a provider")
	})

	t.Run("update capacity minimum", func(t *testing.T) {
		_, err := ms.ProviderUpdateCapacityMinimum(wctx, &types.MsgProviderUpdateCapacityMinimum{
			Creator:            player.Creator,
			ProviderId:         phantom,
			NewMinimumCapacity: 5,
		})
		require.Error(t, err)
	})

	t.Run("withdraw balance", func(t *testing.T) {
		_, err := ms.ProviderWithdrawBalance(wctx, &types.MsgProviderWithdrawBalance{
			Creator:            player.Creator,
			ProviderId:         phantom,
			DestinationAddress: acc.String(),
		})
		require.Error(t, err)
	})

	t.Run("delete", func(t *testing.T) {
		_, err := ms.ProviderDelete(wctx, &types.MsgProviderDelete{
			Creator:    player.Creator,
			ProviderId: phantom,
		})
		require.Error(t, err)
	})

	t.Run("a well-formed provider id that names nothing", func(t *testing.T) {
		_, err := ms.ProviderUpdateAccessPolicy(wctx, &types.MsgProviderUpdateAccessPolicy{
			Creator:      player.Creator,
			ProviderId:   keeperlib.GetObjectID(types.ObjectType_provider, 9999),
			AccessPolicy: types.ProviderAccessPolicy_openMarket,
		})
		require.Error(t, err, "the right namespace is not the same as an existing record")
	})

	// Nothing was written under any of those ids, and in particular nothing
	// under the empty key the phantom commit would have used.
	t.Run("no provider was created", func(t *testing.T) {
		require.Equal(t, providersBefore, k.GetProviderCount(ctx))

		_, found := k.GetProvider(ctx, phantom)
		require.False(t, found)

		_, found = k.GetProvider(ctx, "")
		require.False(t, found)
	})
}
