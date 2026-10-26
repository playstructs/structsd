package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgProviderDelete(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		ownership := "player owned"
		if guildOwned {
			ownership = "guild owned"
		}
		t.Run(ownership, func(t *testing.T) {
			for _, name := range []string{"owner with exact key bit", "explicit delegate", "missing provider", "unregistered signer", "unprivileged player", "owner key lacks delete", "delegate key lacks delete"} {
				t.Run(name, func(t *testing.T) {
					f := setupTeardownFixture(t, 10, "0", "0")
					ctx := sdk.UnwrapSDKContext(f.ctx)
					if guildOwned {
						guild := types.CreateEmptyGuild()
						guild.Id, guild.Index, guild.Owner = "0-900", 900, f.provider.Owner
						f.k.SetGuild(ctx, guild)
						f.provider.Owner = guild.Id
						_, err := f.k.SetProvider(ctx, f.provider)
						require.NoError(t, err)
					}
					address := sdk.AccAddress("delete_other_player_")
					other := testAppendPlayer(f.k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
					msg := &types.MsgProviderDelete{Creator: f.provider.Creator, ProviderId: f.provider.Id}
					f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermDelete)
					f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(other.Creator), types.PermDelete)
					wantError := ""
					switch name {
					case "explicit delegate", "delegate key lacks delete":
						msg.Creator = other.Creator
						f.k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(f.provider.Id, other.Id), types.PermDelete)
						if name == "delegate key lacks delete" {
							f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermAll &^ types.PermDelete)
							wantError = "permission"
						}
					case "missing provider":
						msg.ProviderId, wantError = "10-999999", "not found"
					case "unregistered signer":
						msg.Creator, wantError = sdk.AccAddress("delete_unknown_key__").String(), "player"
					case "unprivileged player":
						msg.Creator, wantError = other.Creator, "permission"
					case "owner key lacks delete":
						f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermAll &^ types.PermDelete)
						wantError = "permission"
					}
					f.fund(t, f.earningsAcc, 7)
					f.fund(t, f.collateralAcc, 11)
					vault, err := types.VaultAddress(f.provider.Owner)
					require.NoError(t, err)
					bank := f.k.BankKeeper().(*keepertest.MockBankKeeper)
					bankBefore := bank.Snapshot()
					gridBefore := f.k.GetAllGridExport(ctx)
					permissionsBefore := f.k.GetAllPermissionExport(ctx)
					ctx = ctx.WithEventManager(sdk.NewEventManager())
					response, err := f.ms.ProviderDelete(ctx, msg)
					require.NotNil(t, response)
					provider, found := f.k.GetProvider(ctx, f.provider.Id)
					if wantError != "" {
						require.ErrorContains(t, err, wantError)
						require.True(t, found)
						require.Equal(t, f.provider, provider)
						require.Equal(t, bankBefore, bank.Snapshot())
						require.Equal(t, gridBefore, f.k.GetAllGridExport(ctx))
						require.Equal(t, permissionsBefore, f.k.GetAllPermissionExport(ctx))
					} else {
						require.NoError(t, err)
						require.False(t, found)
						require.Equal(t, math.NewInt(18), f.balance(vault), "the asset owner receives both pools, even when a delegate deletes")
						require.True(t, f.balance(f.earningsAcc).IsZero())
						require.True(t, f.balance(f.collateralAcc).IsZero())
						owner, associated := f.k.GetVaultOwner(ctx, vault)
						require.True(t, associated)
						require.Equal(t, f.provider.Owner, owner)
						require.Equal(t, types.Permissionless, f.k.GetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(f.provider.Id, other.Id)))
						_, err = f.ms.ProviderDelete(ctx, msg)
						require.ErrorContains(t, err, "not found")
						require.Equal(t, math.NewInt(18), f.balance(vault), "deletion replay cannot pay again")
					}
					deletions := 0
					for _, event := range ctx.EventManager().ABCIEvents() {
						if event.Type == "structs.structs.EventDelete" {
							decoded, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							if decoded.(*types.EventDelete).ObjectId == f.provider.Id {
								deletions++
							}
						}
					}
					if wantError == "" {
						require.Equal(t, 1, deletions)
					} else {
						require.Zero(t, deletions)
					}
				})
			}
		})
	}
}
