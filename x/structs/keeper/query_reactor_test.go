package keeper_test

/*

func TestReactorQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNReactor(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetReactorRequest
		response *types.QueryGetReactorResponse
		err      error
	}{
		{
			desc:    "First",
			request: &types.QueryGetReactorRequest{Id: msgs[0].Id},
			response: &types.QueryGetReactorResponse{
				Reactor:        msgs[0],
				GridAttributes: &types.GridAttributes{},
			},
		},
		{
			desc:    "Second",
			request: &types.QueryGetReactorRequest{Id: msgs[1].Id},
			response: &types.QueryGetReactorResponse{
				Reactor:        msgs[1],
				GridAttributes: &types.GridAttributes{},
			},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetReactorRequest{Id: "non-existent"},
			err:     types.ErrObjectNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Reactor(wctx, tc.request)
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

func TestReactorQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNReactor(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllReactorRequest {
		return &types.QueryAllReactorRequest{
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
			resp, err := keeper.ReactorAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Reactor), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Reactor),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ReactorAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Reactor), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Reactor),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.ReactorAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Reactor),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.ReactorAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
*/
