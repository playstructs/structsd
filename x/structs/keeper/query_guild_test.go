package keeper_test

/*
func TestGuildQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNGuild(keeper, ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetGuildRequest
		response *types.QueryGetGuildResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetGuildRequest{Id: msgs[0].Id},
			response: &types.QueryGetGuildResponse{Guild: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetGuildRequest{Id: msgs[1].Id},
			response: &types.QueryGetGuildResponse{Guild: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetGuildRequest{Id: "nonexistent"},
			err:     types.ErrObjectNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Guild(wctx, tc.request)
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

func TestGuildQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNGuild(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllGuildRequest {
		return &types.QueryAllGuildRequest{
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
			resp, err := keeper.GuildAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Guild), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Guild),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.GuildAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Guild), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Guild),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.GuildAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Guild),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.GuildAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestGuildMembershipApplicationQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test data
	guildId := "guild1"
	playerId := "player1"
	application := createTestGuildMembershipApplication(keeper, ctx, guildId, playerId)

	tests := []struct {
		desc     string
		request  *types.QueryGetGuildMembershipApplicationRequest
		response *types.QueryGetGuildMembershipApplicationResponse
		err      error
	}{
		{
			desc:     "Found",
			request:  &types.QueryGetGuildMembershipApplicationRequest{GuildId: guildId, PlayerId: playerId},
			response: &types.QueryGetGuildMembershipApplicationResponse{GuildMembershipApplication: application},
		},
		{
			desc:    "NotFound",
			request: &types.QueryGetGuildMembershipApplicationRequest{GuildId: "nonexistent", PlayerId: "nonexistent"},
			err:     types.ErrObjectNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.GuildMembershipApplication(wctx, tc.request)
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

func TestGuildBankCollateralAddressQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test guild
	guild := createNGuild(keeper, ctx, 1)[0]

	tests := []struct {
		desc     string
		request  *types.QueryGetGuildBankCollateralAddressRequest
		response *types.QueryAllGuildBankCollateralAddressResponse
		err      error
	}{
		{
			desc:    "Found",
			request: &types.QueryGetGuildBankCollateralAddressRequest{GuildId: guild.Id},
			response: &types.QueryAllGuildBankCollateralAddressResponse{
				InternalAddressAssociation: []*types.InternalAddressAssociation{
					{
						Address:  types.GuildBankCollateralPool + guild.Id,
						ObjectId: guild.Id,
					},
				},
			},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.GuildBankCollateralAddress(wctx, tc.request)
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

func TestGuildBankCollateralAddressAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test guilds
	guilds := createNGuild(keeper, ctx, 3)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllGuildBankCollateralAddressRequest {
		return &types.QueryAllGuildBankCollateralAddressRequest{
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
		for i := 0; i < len(guilds); i += step {
			resp, err := keeper.GuildBankCollateralAddressAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.InternalAddressAssociation), step)
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.GuildBankCollateralAddressAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(guilds), int(resp.Pagination.Total))
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.GuildBankCollateralAddressAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func createTestGuildMembershipApplication(keeper keeper.Keeper, ctx context.Context, guildId string, playerId string) types.GuildMembershipApplication {
	application := types.GuildMembershipApplication{
		GuildId:            guildId,
		PlayerId:           playerId,
		JoinType:           types.GuildJoinType_request,
		RegistrationStatus: types.RegistrationStatus_proposed,
		Proposer:           playerId,
		SubstationId:       "substation1",
	}
	keeper.SetGuildMembershipApplication(ctx, application)
	return application
}
*/
