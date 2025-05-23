package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestAgreementQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.Agreement(ctx, nil)
	require.Error(t, err)

	// Test non-existent agreement
	req := &types.QueryGetAgreementRequest{
		Id: "non-existent",
	}
	_, err = keeper.Agreement(ctx, req)
	require.Error(t, err)

	// Test existing agreement
	agreements := createNAgreement(keeper, ctx, 1)
	agreement := agreements[0]

	req = &types.QueryGetAgreementRequest{
		Id: agreement.Id,
	}
	resp, err := keeper.Agreement(ctx, req)
	require.NoError(t, err)
	require.Equal(t,
		agreement.Id, resp.Agreement.Id,
	)
}

func TestAgreementAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.AgreementAll(ctx, nil)
	require.Error(t, err)

	// Create multiple agreements
	agreements := createNAgreement(keeper, ctx, 5)

	// Test pagination
	req := &types.QueryAllAgreementRequest{
		Pagination: &query.PageRequest{
			Limit: 2,
		},
	}
	resp, err := keeper.AgreementAll(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Agreement, 2)
	require.NotNil(t, resp.Pagination)

	// Test without pagination
	req = &types.QueryAllAgreementRequest{}
	resp, err = keeper.AgreementAll(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Agreement, len(agreements))
	require.Nil(t, resp.Pagination)

	// Verify all agreements are present
	for _, agreement := range agreements {
		found := false
		for _, respAgreement := range resp.Agreement {
			if agreement.Id == respAgreement.Id {
				found = true
				require.Equal(t,
					agreement.Id, respAgreement.Id,
				)
				break
			}
		}
		require.True(t, found)
	}
}

func TestAgreementAllByProviderQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.AgreementAllByProvider(ctx, nil)
	require.Error(t, err)

	// Create agreements for different providers
	provider1Agreements := []types.Agreement{
		{
			Id:         "agreement1",
			ProviderId: "provider1",
			EndBlock:   1000,
		},
		{
			Id:         "agreement2",
			ProviderId: "provider1",
			EndBlock:   2000,
		},
	}
	provider2Agreements := []types.Agreement{
		{
			Id:         "agreement3",
			ProviderId: "provider2",
			EndBlock:   1500,
		},
	}

	// Store agreements
	for _, agreement := range provider1Agreements {
		keeper.AppendAgreement(ctx, agreement)
	}
	for _, agreement := range provider2Agreements {
		keeper.AppendAgreement(ctx, agreement)
	}

	// Test query for provider1
	req := &types.QueryAllAgreementByProviderRequest{
		ProviderId: "provider1",
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err := keeper.AgreementAllByProvider(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Agreement, len(provider1Agreements))
	for _, agreement := range resp.Agreement {
		require.Equal(t, "provider1", agreement.ProviderId)
	}

	// Test query for provider2
	req = &types.QueryAllAgreementByProviderRequest{
		ProviderId: "provider2",
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err = keeper.AgreementAllByProvider(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Agreement, len(provider2Agreements))
	for _, agreement := range resp.Agreement {
		require.Equal(t, "provider2", agreement.ProviderId)
	}

	// Test query for non-existent provider
	req = &types.QueryAllAgreementByProviderRequest{
		ProviderId: "non-existent",
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err = keeper.AgreementAllByProvider(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Agreement, 0)
}
