package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgProviderUpdateCapacityMinimum(t *testing.T) {
	testProviderBoundUpdate(t, false, false)
}

func testProviderBoundUpdate(t *testing.T, maximum, duration bool) {
	for _, guildOwned := range []bool{false, true} {
		ownership := "player owned"
		if guildOwned {
			ownership = "guild owned"
		}
		t.Run(ownership, func(t *testing.T) {
			for _, name := range []string{"owner with exact key bit", "explicit delegate", "missing provider", "unregistered signer", "unprivileged player", "key lacks update", "zero bound", "equal opposite bound", "crossed range", "maximum uint64", "unchanged bound"} {
				t.Run(name, func(t *testing.T) {
					k, ms, goCtx := setupMsgServer(t)
					ctx := sdk.UnwrapSDKContext(goCtx)
					gs := testCreateGuild(k, ctx)
					address := sdk.AccAddress("provider_range_actor")
					other := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
					ownerID := gs.GuildOwner.Id
					if guildOwned {
						ownerID = gs.Guild.Id
					}
					provider := testAppendProvider(k, ctx, types.Provider{
						Owner: ownerID, Creator: gs.GuildOwner.Creator, SubstationId: gs.Guild.EntrySubstationId,
						Rate: sdk.NewCoin("token", math.NewInt(100)), AccessPolicy: types.ProviderAccessPolicy_closedMarket,
						CapacityMinimum: 100, CapacityMaximum: 1000, DurationMinimum: 100, DurationMaximum: 1000,
						ProviderCancellationPenalty: math.LegacyNewDec(1), ConsumerCancellationPenalty: math.LegacyNewDec(1),
					})
					creator, providerID, wantError := gs.GuildOwner.Creator, provider.Id, ""
					value, expectedValue := uint64(50), uint64(50)
					if maximum {
						value, expectedValue = 2000, 2000
					}
					k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, provider.Id), 400)
					k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, provider.Id), 7)
					ctx = ctx.WithBlockHeight(20)
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermUpdate)
					switch name {
					case "explicit delegate":
						creator = other.Creator
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(provider.Id, other.Id), types.PermUpdate)
						k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermUpdate)
					case "missing provider":
						providerID, wantError = "11-999999", "not found"
					case "unregistered signer":
						creator, wantError = sdk.AccAddress("provider_unknown_key").String(), "player"
					case "unprivileged player":
						creator, wantError = other.Creator, "permission"
					case "key lacks update":
						k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermAll&^types.PermUpdate)
						wantError = "permission"
					case "zero bound":
						value, expectedValue = 0, 1
						if maximum {
							wantError = "cannot be lower than minimum"
						}
					case "equal opposite bound":
						value, expectedValue = 1000, 1000
						if maximum {
							value, expectedValue = 100, 100
						}
					case "crossed range":
						value, wantError = 1001, "exceeds_maximum"
						if maximum {
							value, wantError = 99, "cannot be lower than minimum"
						}
					case "maximum uint64":
						value, expectedValue = ^uint64(0), ^uint64(0)
						if !maximum {
							wantError = "exceeds_maximum"
						}
					case "unchanged bound":
						value, expectedValue = 100, 100
						if maximum {
							value, expectedValue = 1000, 1000
						}
					}
					permissions := k.GetAllPermissionExport(ctx)
					grid := k.GetAllGridExport(ctx)
					ctx = ctx.WithEventManager(sdk.NewEventManager())
					var response *types.MsgProviderResponse
					var err error
					if duration && maximum {
						response, err = ms.ProviderUpdateDurationMaximum(ctx, &types.MsgProviderUpdateDurationMaximum{Creator: creator, ProviderId: providerID, NewMaximumDuration: value})
					} else if duration {
						response, err = ms.ProviderUpdateDurationMinimum(ctx, &types.MsgProviderUpdateDurationMinimum{Creator: creator, ProviderId: providerID, NewMinimumDuration: value})
					} else if maximum {
						response, err = ms.ProviderUpdateCapacityMaximum(ctx, &types.MsgProviderUpdateCapacityMaximum{Creator: creator, ProviderId: providerID, NewMaximumCapacity: value})
					} else {
						response, err = ms.ProviderUpdateCapacityMinimum(ctx, &types.MsgProviderUpdateCapacityMinimum{Creator: creator, ProviderId: providerID, NewMinimumCapacity: value})
					}
					require.NotNil(t, response)
					actual, found := k.GetProvider(ctx, provider.Id)
					require.True(t, found)
					expected := provider
					if wantError != "" {
						require.ErrorContains(t, err, wantError)
					} else {
						require.NoError(t, err)
						if duration && maximum {
							expected.DurationMaximum = expectedValue
						} else if duration {
							expected.DurationMinimum = expectedValue
						} else if maximum {
							expected.CapacityMaximum = expectedValue
						} else {
							expected.CapacityMinimum = expectedValue
						}
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
