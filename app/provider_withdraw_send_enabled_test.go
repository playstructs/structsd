package app_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Provider earnings withdrawals are voluntary transfers. Neither choosing an
// external recipient nor omitting it in favor of a vault may bypass a denom
// freeze. Use real bank/auth stores: the mock's balances do not roll back with
// the handler's SDK cache, including settlement performed before withdrawal.
func TestProviderWithdrawalHonorsSendEnabled(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, external := range []bool{false, true} {
			for _, pending := range []int{0, 1} {
				t.Run(fmt.Sprintf("guild=%t/external=%t/pending=%d", guildOwned, external, pending), func(t *testing.T) {
					bApp, ctx, providerID, ids, consumers := setupFundedAgreements(t, pending)
					provider, found := bApp.StructsKeeper.GetProvider(ctx, providerID)
					require.True(t, found)
					if guildOwned {
						guild := types.CreateEmptyGuild()
						guild.Id, guild.Index, guild.Owner, guild.Creator = "0-991", 991, provider.Owner, provider.Creator
						bApp.StructsKeeper.SetGuild(ctx, guild)
						provider.Owner = guild.Id
						_, err := bApp.StructsKeeper.SetProvider(ctx, provider)
						require.NoError(t, err)
					}
					if pending != 0 {
						ctx = ctx.WithBlockHeight(3)
						cc := bApp.StructsKeeper.NewCurrentContext(ctx)
						require.NoError(t, cc.GetAgreement(ids[0]).PrematureCloseByConsumer())
						cc.CommitAll()
					}
					earnings := keeperlib.GetProviderEarningsPoolLocation(providerID)
					coins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 7), sdk.NewInt64Coin("ufrozen", 5))
					fundRedelegationAccount(t, bApp, ctx, earnings, coins)
					vault, err := types.VaultAddress(provider.Owner)
					require.NoError(t, err)
					recipient := vault
					msg := &types.MsgProviderWithdrawBalance{Creator: provider.Creator, ProviderId: providerID}
					if external {
						recipient, _, _ = newTransferPlayer(t, bApp, ctx)
						msg.DestinationAddress = recipient.String()
					}
					bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(provider.Creator), types.PermProviderWithdraw)
					bApp.BankKeeper.SetSendEnabled(ctx, "ufrozen", false)
					ctx = ctx.WithEventManager(sdk.NewEventManager())
					before := applicationAssetStateRows(bApp, ctx)
					server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
					_, err = server.ProviderWithdrawBalance(ctx, msg)
					require.ErrorContains(t, err, "send transactions are disabled", "a frozen withdrawal must fail: recipient=%s pool=%s", bApp.BankKeeper.GetBalance(ctx, recipient, "ufrozen"), bApp.BankKeeper.GetBalance(ctx, earnings, "ufrozen"))
					require.Equal(t, before, applicationAssetStateRows(bApp, ctx), "refusal rolls back settlement, balances, accounts, associations and checkpoint state")
					require.Empty(t, ctx.EventManager().ABCIEvents(), "a rejected withdrawal publishes no partial settlement or asset events")
					bApp.BankKeeper.SetSendEnabled(ctx, "ufrozen", true)
					for pass := 0; pass < 5; pass++ {
						_, err = server.ProviderWithdrawBalance(ctx, msg)
						require.NoError(t, err)
					}
					wantAlpha := int64(7)
					if pending != 0 {
						wantAlpha += 8
						require.Equal(t, int64(12), bApp.BankKeeper.GetBalance(ctx, consumers[0], "ualpha").Amount.Int64())
						_, live := bApp.StructsKeeper.GetAgreement(ctx, ids[0])
						require.False(t, live)
						require.Equal(t, int64(1), bApp.BankKeeper.GetBalance(ctx, keeperlib.GetProviderCollateralPoolLocation(providerID), "ualpha").Amount.Int64(), "ordinary checkpoint/penalty rounding leaves one recoverable dust coin")
					}
					require.Equal(t, int64(5), bApp.BankKeeper.GetBalance(ctx, recipient, "ufrozen").Amount.Int64())
					require.Equal(t, wantAlpha, bApp.BankKeeper.GetBalance(ctx, recipient, "ualpha").Amount.Int64())
					require.Empty(t, bApp.BankKeeper.GetAllBalances(ctx, earnings))
					if !external {
						owner, associated := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
						require.True(t, associated)
						require.Equal(t, provider.Owner, owner)
					}
				})
			}
		}
	}
}
