package keeper_test

/* Cannot perform test because account keeper is not implemented
func TestProviderQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProvider(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetProviderRequest
		response *types.QueryGetProviderResponse
		err      error
	}{
		{
			desc:    "First",
			request: &types.QueryGetProviderRequest{Id: msgs[0].Id},
			response: &types.QueryGetProviderResponse{
				Provider:       msgs[0],
				GridAttributes: &types.GridAttributes{},
			},
		},
		{
			desc:    "Second",
			request: &types.QueryGetProviderRequest{Id: msgs[1].Id},
			response: &types.QueryGetProviderResponse{
				Provider:       msgs[1],
				GridAttributes: &types.GridAttributes{},
			},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetProviderRequest{Id: "non-existent"},
			err:     types.ErrObjectNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Provider(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestProviderQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProvider(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllProviderRequest {
		return &types.QueryAllProviderRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ProviderAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Provider), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Provider),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ProviderAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Provider), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Provider),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.ProviderAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Provider),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.ProviderAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestProviderCollateralAddressQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProvider(keeper, ctx, 2)

	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetProviderCollateralAddressRequest
		response *types.QueryAllProviderCollateralAddressResponse
		err      error
	}{
		{
			desc:    "First",
			request: &types.QueryGetProviderCollateralAddressRequest{ProviderId: msgs[0].Id},
			response: &types.QueryAllProviderCollateralAddressResponse{
				InternalAddressAssociation: []*types.InternalAddressAssociation{
					{
						Address:  types.ProviderCollateralPool + msgs[0].Id,
						ObjectId: msgs[0].Id,
					},
				},
			},
		},
		{
			desc:    "Second",
			request: &types.QueryGetProviderCollateralAddressRequest{ProviderId: msgs[1].Id},
			response: &types.QueryAllProviderCollateralAddressResponse{
				InternalAddressAssociation: []*types.InternalAddressAssociation{
					{
						Address:  types.ProviderCollateralPool + msgs[1].Id,
						ObjectId: msgs[1].Id,
					},
				},
			},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.ProviderCollateralAddress(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestProviderCollateralAddressAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProvider(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllProviderCollateralAddressRequest {
		return &types.QueryAllProviderCollateralAddressRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ProviderCollateralAddressAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.InternalAddressAssociation), step)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ProviderCollateralAddressAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.InternalAddressAssociation), step)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.ProviderCollateralAddressAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.ProviderCollateralAddressAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestProviderEarningsAddressQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProvider(keeper, ctx, 2)

	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetProviderEarningsAddressRequest
		response *types.QueryAllProviderEarningsAddressResponse
		err      error
	}{
		{
			desc:    "First",
			request: &types.QueryGetProviderEarningsAddressRequest{ProviderId: msgs[0].Id},
			response: &types.QueryAllProviderEarningsAddressResponse{
				InternalAddressAssociation: []*types.InternalAddressAssociation{
					{
						Address:  types.ProviderEarningsPool + msgs[0].Id,
						ObjectId: msgs[0].Id,
					},
				},
			},
		},
		{
			desc:    "Second",
			request: &types.QueryGetProviderEarningsAddressRequest{ProviderId: msgs[1].Id},
			response: &types.QueryAllProviderEarningsAddressResponse{
				InternalAddressAssociation: []*types.InternalAddressAssociation{
					{
						Address:  types.ProviderEarningsPool + msgs[1].Id,
						ObjectId: msgs[1].Id,
					},
				},
			},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.ProviderEarningsAddress(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestProviderEarningsAddressAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProvider(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllProviderEarningsAddressRequest {
		return &types.QueryAllProviderEarningsAddressRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ProviderEarningsAddressAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.InternalAddressAssociation), step)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ProviderEarningsAddressAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.InternalAddressAssociation), step)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.ProviderEarningsAddressAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.ProviderEarningsAddressAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
*/
