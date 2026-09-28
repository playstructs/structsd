package app_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	structstypes "structs/x/structs/types"
)

func TestConsumerHoldbackSettlementWithRealBankRestriction(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		name := "player provider"
		if guildOwned {
			name = "guild provider"
		}
		t.Run(name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			first := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x71}, 20)))
			second := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x72}, 20)))
			owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x73}, 20)))
			guild := structstypes.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner = "0-1900", 1900, owner.Id
			bApp.StructsKeeper.SetGuild(ctx, guild)
			ownerID := owner.Id
			if guildOwned {
				ownerID = guild.Id
			}
			denom := "uguild." + guild.Id
			station := structstypes.Substation{Id: keeperlib.GetObjectID(structstypes.ObjectType_substation, 1900), Owner: owner.Id, Creator: owner.Creator}
			bApp.StructsKeeper.SetSubstation(ctx, station)
			bApp.StructsKeeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(structstypes.GridAttributeType_capacity, owner.Id), 1000)
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			_, supplyErr := cc.NewAllocation(structstypes.AllocationType_automated, owner.Id, station.Id, owner.Creator, owner.Id, 0)
			require.NoError(t, supplyErr)
			provider := cc.NewProvider(structstypes.Provider{
				Owner: ownerID, Creator: owner.Creator, SubstationId: station.Id,
				Rate: sdk.NewInt64Coin(denom, 10), AccessPolicy: structstypes.ProviderAccessPolicy_openMarket,
				CapacityMinimum: 1, CapacityMaximum: 1000, DurationMinimum: 1, DurationMaximum: 1000,
				ProviderCancellationPenalty: math.LegacyNewDecWithPrec(5, 1), ConsumerCancellationPenalty: math.LegacyNewDecWithPrec(25, 2),
			})
			cc.CommitAll()
			firstVaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(first.Id)
			require.NoError(t, err)
			secondVaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(second.Id)
			require.NoError(t, err)
			ownerVaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(ownerID)
			require.NoError(t, err)
			require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(denom, 150000))))
			for i, consumer := range []structstypes.Player{first, second} {
				vault := firstVaults[0]
				capacity := uint64(100)
				if i == 1 {
					vault, capacity = secondVaults[0], 200
				}
				require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, vault, sdk.NewCoins(sdk.NewInt64Coin(denom, int64(capacity)*500))))
				_, err := ms.AgreementOpen(ctx, &structstypes.MsgAgreementOpen{Creator: consumer.Creator, ProviderId: provider.ID(), Capacity: capacity, Duration: 50})
				require.NoError(t, err)
			}
			agreements := bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(ctx, provider.ID())
			require.Len(t, agreements, 2)
			byOwner := map[string]string{}
			for _, id := range agreements {
				agreement, found := bApp.StructsKeeper.GetAgreement(ctx, id)
				require.True(t, found)
				byOwner[agreement.Owner] = id
			}
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 20).WithEventManager(sdk.NewEventManager())
			collateral := keeperlib.GetProviderCollateralPoolLocation(provider.ID())
			earnings := keeperlib.GetProviderEarningsPoolLocation(provider.ID())
			// Checkpoint has already moved revenue when a refund fails. Model the
			// SDK transaction cache and prove none of that partial work escapes.
			rejectRefund := true
			refusedRefunds := 0
			bApp.BankKeeper.AppendSendRestriction(func(_ context.Context, from, to sdk.AccAddress, _ sdk.Coins) (sdk.AccAddress, error) {
				if rejectRefund && from.Equals(collateral) && to.Equals(firstVaults[0]) {
					refusedRefunds++
					return to, errors.New("injected consumer refund refusal")
				}
				return to, nil
			})
			gridBefore := bApp.StructsKeeper.GetAllGridExport(ctx)
			permissionsBefore := bApp.StructsKeeper.GetAllPermissionExport(ctx)
			failedCtx, _ := ctx.CacheContext()
			_, err = ms.AgreementClose(failedCtx, &structstypes.MsgAgreementClose{Creator: first.Creator, AgreementId: byOwner[first.Id]})
			require.ErrorContains(t, err, "collateral payout of 22500"+denom+" failed")
			require.Equal(t, 1, refusedRefunds)
			require.Equal(t, int64(30000), bApp.BankKeeper.GetBalance(failedCtx, earnings, denom).Amount.Int64(), "exercise failure after checkpoint revenue moved")
			require.Equal(t, int64(120000), bApp.BankKeeper.GetBalance(failedCtx, collateral, denom).Amount.Int64())
			require.True(t, bApp.BankKeeper.GetBalance(ctx, earnings, denom).IsZero())
			require.Equal(t, int64(150000), bApp.BankKeeper.GetBalance(ctx, collateral, denom).Amount.Int64())
			require.True(t, bApp.BankKeeper.GetBalance(ctx, firstVaults[0], denom).IsZero())
			require.True(t, bApp.BankKeeper.GetBalance(ctx, secondVaults[0], denom).IsZero())
			require.Equal(t, gridBefore, bApp.StructsKeeper.GetAllGridExport(ctx))
			require.Equal(t, permissionsBefore, bApp.StructsKeeper.GetAllPermissionExport(ctx))
			require.Equal(t, agreements, bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(ctx, provider.ID()))
			for _, id := range agreements {
				agreement, found := bApp.StructsKeeper.GetAgreement(ctx, id)
				require.True(t, found)
				_, found = bApp.StructsKeeper.GetAllocation(ctx, agreement.AllocationId)
				require.True(t, found)
				require.Contains(t, bApp.StructsKeeper.GetAllAgreementIdByExpirationIndex(ctx, agreement.EndBlock), id)
			}
			require.Empty(t, ctx.EventManager().Events(), "discarded settlement events must not reach indexers")
			rejectRefund = false
			_, err = ms.AgreementClose(ctx, &structstypes.MsgAgreementClose{Creator: first.Creator, AgreementId: byOwner[first.Id]})
			require.NoError(t, err)
			require.Equal(t, int64(22500), bApp.BankKeeper.GetBalance(ctx, firstVaults[0], denom).Amount.Int64())
			require.True(t, bApp.BankKeeper.GetBalance(ctx, secondVaults[0], denom).IsZero())
			require.Equal(t, int64(80000), bApp.BankKeeper.GetBalance(ctx, collateral, denom).Amount.Int64())
			require.Equal(t, int64(47500), bApp.BankKeeper.GetBalance(ctx, earnings, denom).Amount.Int64())
			_, err = ms.AgreementClose(ctx, &structstypes.MsgAgreementClose{Creator: second.Creator, AgreementId: byOwner[second.Id]})
			require.NoError(t, err)
			_, err = ms.ProviderWithdrawBalance(ctx, &structstypes.MsgProviderWithdrawBalance{Creator: owner.Creator, ProviderId: provider.ID()})
			require.NoError(t, err)
			for _, balance := range []struct {
				address sdk.AccAddress
				amount  int64
			}{
				{firstVaults[0], 22500}, {secondVaults[0], 45000}, {ownerVaults[0], 82500}, {collateral, 0}, {earnings, 0},
			} {
				require.Equal(t, balance.amount, bApp.BankKeeper.GetBalance(ctx, balance.address, denom).Amount.Int64())
			}
			require.Equal(t, int64(150000), bApp.BankKeeper.GetSupply(ctx, denom).Amount.Int64())
			message, broken := keeperlib.ProviderCollateralSolvencyInvariant(bApp.StructsKeeper)(ctx)
			require.False(t, broken, message)
			refunds := map[string]int{}
			withdrawals := 0
			for _, event := range ctx.EventManager().Events() {
				if event.Type != "transfer" {
					continue
				}
				attributes := map[string]string{}
				for _, attribute := range event.Attributes {
					attributes[attribute.Key] = attribute.Value
				}
				if attributes["recipient"] == ownerVaults[0].String() {
					require.Equal(t, earnings.String(), attributes["sender"])
					require.Equal(t, "82500"+denom, attributes["amount"])
					withdrawals++
				}
				for i, vault := range []sdk.AccAddress{firstVaults[0], secondVaults[0]} {
					if attributes["recipient"] == vault.String() {
						require.Equal(t, collateral.String(), attributes["sender"])
						require.Equal(t, []string{"22500", "45000"}[i]+denom, attributes["amount"])
						refunds[vault.String()]++
					}
				}
			}
			require.Equal(t, map[string]int{firstVaults[0].String(): 1, secondVaults[0].String(): 1}, refunds)
			require.Equal(t, 1, withdrawals)
		})
	}
}
