package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"
)

func TestDiversityQueryValidationAndReadOnlyFailures(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	_, _, owner := newTransferPlayer(t, bApp, ctx)
	for _, sample := range []struct {
		name    string
		request *types.QueryOwnerReactorDiversityRequest
		code    codes.Code
	}{
		{"nil", nil, codes.InvalidArgument},
		{"empty", &types.QueryOwnerReactorDiversityRequest{}, codes.InvalidArgument},
		{"unsupported-object", &types.QueryOwnerReactorDiversityRequest{OwnerId: "3-1"}, codes.InvalidArgument},
		{"noncanonical", &types.QueryOwnerReactorDiversityRequest{OwnerId: "1-01"}, codes.InvalidArgument},
		{"absent-player", &types.QueryOwnerReactorDiversityRequest{OwnerId: "1-999999"}, codes.NotFound},
		{"absent-guild", &types.QueryOwnerReactorDiversityRequest{OwnerId: "0-999999"}, codes.NotFound},
	} {
		t.Run(sample.name, func(t *testing.T) {
			before := applicationAssetStateRows(bApp, ctx)
			events := len(ctx.EventManager().Events())
			response, err := bApp.StructsKeeper.OwnerReactorDiversity(ctx, sample.request)
			require.Nil(t, response)
			require.Equal(t, sample.code, status.Code(err))
			require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
			require.Len(t, ctx.EventManager().Events(), events)
		})
	}
	for _, summary := range []types.OwnerReactorDiversity{
		{OwnerId: owner, Power: "01", SquaredPower: "1"},
		{OwnerId: owner, Power: "1", SquaredPower: "-1"},
		{OwnerId: owner, Power: "invalid", SquaredPower: "1"},
		{OwnerId: "1-999999", Power: "1", SquaredPower: "1"},
	} {
		branch, _ := ctx.CacheContext()
		branch.KVStore(bApp.GetKey(types.StoreKey)).Set([]byte("ReactorDiversity/owner/"+owner), bApp.AppCodec().MustMarshal(&summary))
		before := applicationAssetStateRows(bApp, branch)
		events := len(branch.EventManager().Events())
		response, err := bApp.StructsKeeper.OwnerReactorDiversity(branch, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
		require.Nil(t, response)
		require.Equal(t, codes.Internal, status.Code(err), "%+v", summary)
		require.Equal(t, before, applicationAssetStateRows(bApp, branch))
		require.Len(t, branch.EventManager().Events(), events)
	}
	// An existing owner without stake is distinct from a missing owner or an
	// incomplete inventory. Reading it must not create a derived summary row.
	before := applicationAssetStateRows(bApp, ctx)
	events := len(ctx.EventManager().Events())
	response, err := bApp.StructsKeeper.OwnerReactorDiversity(ctx, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
	require.NoError(t, err)
	require.Equal(t, &types.OwnerReactorDiversity{OwnerId: owner, Power: "0", SquaredPower: "0"}, response.Stored)
	require.True(t, response.Initialized)
	require.True(t, response.LivePortfolioComplete)
	require.True(t, response.InfusionInventoryComplete)
	require.Equal(t, "0", response.LiveBonus)
	require.Zero(t, response.UsableBonus)
	require.Zero(t, response.UsablePrimaryCapacity)
	_, exists := bApp.StructsKeeper.GetOwnerReactorDiversity(ctx, owner)
	require.False(t, exists)
	require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
	require.Len(t, ctx.EventManager().Events(), events)
}
