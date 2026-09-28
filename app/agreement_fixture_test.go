package app_test

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"strings"
	"structs/app"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
	"testing"
)

func setupFundedAgreements(t *testing.T, count int) (*app.App, sdk.Context, string, []string, []sdk.AccAddress) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	return setupFundedAgreementsOnApp(t, bApp, ctx, count)
}

func setupFundedAgreementsOnApp(t *testing.T, bApp *app.App, ctx sdk.Context, count int) (*app.App, sdk.Context, string, []string, []sdk.AccAddress) {
	t.Helper()
	return setupFundedAgreementsOnGraph(t, bApp, ctx, count, 0)
}

func setupFundedAgreementsOnGraph(t *testing.T, bApp *app.App, ctx sdk.Context, count, links int) (*app.App, sdk.Context, string, []string, []sdk.AccAddress) {
	t.Helper()
	return setupFundedAgreementsWithDuration(t, bApp, ctx, count, links, 7)
}

func setupFundedAgreementsWithDuration(t *testing.T, bApp *app.App, ctx sdk.Context, count, links int, duration uint64) (*app.App, sdk.Context, string, []string, []sdk.AccAddress) {
	t.Helper()
	primary, _, owner := newTransferPlayer(t, bApp, ctx)
	bApp.StructsKeeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner), 1000)
	const station = "4-97000"
	bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: station, Owner: owner})
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	source := owner
	for i := 0; i < links; i++ {
		next := fmt.Sprintf("4-%08d", 96000+i)
		bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: next, Owner: owner})
		cc.GenesisImportAllocation(types.Allocation{Id: fmt.Sprintf("6-%08d", 96000+i), SourceObjectId: source, DestinationId: next, Type: types.AllocationType_dynamic}, 1000)
		source = next
	}
	_, err := cc.NewAllocation(types.AllocationType_dynamic, source, station, primary.String(), owner, 1000)
	require.NoError(t, err)
	provider := cc.NewProvider(types.Provider{Owner: owner, Creator: primary.String(), SubstationId: station, Rate: sdk.NewInt64Coin("ualpha", 3),
		AccessPolicy: types.ProviderAccessPolicy_openMarket, CapacityMinimum: 1, CapacityMaximum: 1000, DurationMinimum: 1, DurationMaximum: 100,
		ProviderCancellationPenalty: math.LegacyNewDecWithPrec(4, 1), ConsumerCancellationPenalty: math.LegacyNewDecWithPrec(25, 2)})
	cc.CommitAll()
	ids, vaults := []string{}, []sdk.AccAddress{}
	server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	for i := 0; i < count; i++ {
		address, _, consumer := newTransferPlayer(t, bApp, ctx)
		accounts, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(consumer)
		require.NoError(t, err)
		fundRedelegationAccount(t, bApp, ctx, accounts[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", int64(3*duration))))
		_, err = server.AgreementOpen(ctx, &types.MsgAgreementOpen{Creator: address.String(), ProviderId: provider.ID(), Capacity: 1, Duration: duration})
		require.NoError(t, err)
		vaults = append(vaults, accounts[0])
		for _, id := range bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(ctx, provider.ID()) {
			row, _ := bApp.StructsKeeper.GetAgreement(ctx, id)
			if row.Owner == consumer {
				ids = append(ids, id)
			}
		}
	}
	return bApp, ctx, provider.ID(), ids, vaults
}

func applicationAssetStateRows(bApp *app.App, ctx sdk.Context) map[string]string {
	rows := map[string]string{}
	for _, name := range []string{types.StoreKey, banktypes.StoreKey, authtypes.StoreKey} {
		it := storetypes.KVStorePrefixIterator(ctx.KVStore(bApp.GetKey(name)), nil)
		for ; it.Valid(); it.Next() {
			rows[name+"/"+string(it.Key())] = string(it.Value())
		}
		it.Close()
	}
	return rows
}

func bankAndAuthStateRows(bApp *app.App, ctx sdk.Context) map[string]string {
	rows := applicationAssetStateRows(bApp, ctx)
	for key := range rows {
		if strings.HasPrefix(key, types.StoreKey+"/") {
			delete(rows, key)
		}
	}
	return rows
}
