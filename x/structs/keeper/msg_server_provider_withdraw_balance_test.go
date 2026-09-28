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

func TestMsgProviderWithdrawBalance(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		ownership := "player owned"
		if guildOwned {
			ownership = "guild owned"
		}
		t.Run(ownership, func(t *testing.T) {
			for _, name := range []string{"owner with exact key bit", "explicit delegate", "missing provider", "unregistered signer", "unprivileged player", "owner key lacks withdraw", "delegate key lacks withdraw", "invalid recipient"} {
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
					address := sdk.AccAddress("withdraw_other_player")
					other := testAppendPlayer(f.k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
					recipient := sdk.AccAddress("withdraw_recipient__")
					msg := &types.MsgProviderWithdrawBalance{Creator: f.provider.Creator, ProviderId: f.provider.Id, DestinationAddress: recipient.String()}
					f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermProviderWithdraw)
					f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(other.Creator), types.PermProviderWithdraw)
					wantError := ""
					switch name {
					case "explicit delegate", "delegate key lacks withdraw":
						msg.Creator = other.Creator
						f.k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(f.provider.Id, other.Id), types.PermProviderWithdraw)
						if name == "delegate key lacks withdraw" {
							f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermAll &^ types.PermProviderWithdraw)
							wantError = "permission"
						}
					case "missing provider":
						msg.ProviderId, wantError = "10-999999", "not found"
					case "unregistered signer":
						msg.Creator, wantError = sdk.AccAddress("withdraw_unknown_key").String(), "player"
					case "unprivileged player":
						msg.Creator, wantError = other.Creator, "permission"
					case "owner key lacks withdraw":
						f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermAll &^ types.PermProviderWithdraw)
						wantError = "permission"
					case "invalid recipient":
						msg.DestinationAddress, wantError = "invalid-recipient", "address"
					}
					f.fund(t, f.earningsAcc, 7)
					f.fund(t, f.collateralAcc, 11)
					bank := f.k.BankKeeper().(*keepertest.MockBankKeeper)
					bankBefore := bank.Snapshot()
					gridBefore := f.k.GetAllGridExport(ctx)
					permissionsBefore := f.k.GetAllPermissionExport(ctx)
					ctx = ctx.WithEventManager(sdk.NewEventManager())
					response, err := f.ms.ProviderWithdrawBalance(ctx, msg)
					require.NotNil(t, response, "handlers return an empty response with an error")
					if wantError != "" {
						require.ErrorContains(t, err, wantError)
						require.Equal(t, bankBefore, bank.Snapshot())
						require.Equal(t, gridBefore, f.k.GetAllGridExport(ctx))
						require.Empty(t, ctx.EventManager().ABCIEvents())
					} else {
						require.NoError(t, err)
						require.Equal(t, math.NewInt(7), f.balance(recipient))
						require.True(t, f.balance(f.earningsAcc).IsZero())
						_, err = f.ms.ProviderWithdrawBalance(ctx, msg)
						require.NoError(t, err)
						require.Equal(t, math.NewInt(7), f.balance(recipient), "an empty repeat cannot pay twice")
					}
					require.Equal(t, math.NewInt(11), f.balance(f.collateralAcc), "unearned collateral is not withdrawable")
					require.Equal(t, permissionsBefore, f.k.GetAllPermissionExport(ctx))
					provider, found := f.k.GetProvider(ctx, f.provider.Id)
					require.True(t, found)
					require.Equal(t, f.provider, provider)
				})
			}
		})
	}
}

/* TestProviderWithdrawBalanceRejectsBlockedDestination is the second
 * transaction-chosen recipient in this module.
 *
 * PlayerSend gets most of the attention because the recipient is the point of
 * the message, but a withdrawal names where the money goes too, and it reached
 * SendCoins the same way: straight past the blocked-address set, which lives in
 * the bank's MsgServer rather than in SendCoins itself. Both now resolve the
 * destination through resolveExternalRecipient.
 */
func TestProviderWithdrawBalanceRejectsBlockedDestination(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	ownerAcc := sdk.AccAddress("withdraw_blocked_owner_pad_00001")
	owner := testAppendPlayer(k, ctx, types.Player{
		Creator:        ownerAcc.String(),
		PrimaryAddress: ownerAcc.String(),
	})

	sourceObjectId := "withdraw-blocked-source"
	k.SetGridAttribute(ctx,
		keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId), 1000)

	allocation, err := testAppendAllocation(k, ctx, types.Allocation{
		SourceObjectId: sourceObjectId,
		Type:           types.AllocationType_static,
		Controller:     owner.Id,
	}, 100)
	require.NoError(t, err)

	substation, _, err := testAppendSubstation(k, ctx, allocation, owner)
	require.NoError(t, err)

	provider := testAppendProvider(k, ctx, types.Provider{
		Owner:                       owner.Id,
		Creator:                     owner.Creator,
		SubstationId:                substation.Id,
		Rate:                        sdk.NewCoin("token", math.NewInt(100)),
		AccessPolicy:                types.ProviderAccessPolicy_openMarket,
		CapacityMinimum:             100,
		CapacityMaximum:             1000,
		DurationMinimum:             1,
		DurationMaximum:             10,
		ProviderCancellationPenalty: math.LegacyNewDec(1),
		ConsumerCancellationPenalty: math.LegacyNewDec(1),
	})

	blocked := sdk.AccAddress("withdraw_blocked_pool_pad_000001")
	k.BankKeeper().(*keepertest.MockBankKeeper).BlockAddress(blocked)

	_, err = ms.ProviderWithdrawBalance(wctx, &types.MsgProviderWithdrawBalance{
		Creator:            owner.Creator,
		ProviderId:         provider.Id,
		DestinationAddress: blocked.String(),
	})
	require.Error(t, err, "a withdrawal must not reach an account the app declared unreachable")
	require.Contains(t, err.Error(), "blocked_recipient")
}
