package keeper_test

/*
func createNReactor(keeper keeper.Keeper, ctx sdk.Context, n int) []types.Reactor {
	items := make([]types.Reactor, n)
	for i := range items {
		items[i] = keeper.AppendReactor(ctx, items[i])
	}
	return items
}

func TestReactorGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetReactor(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestReactorRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveReactor(ctx, item.Id)
		_, found := keeper.GetReactor(ctx, item.Id)
		require.False(t, found)
	}
}

func TestReactorGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllReactor(ctx)),
	)
}

func TestReactorCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetReactorCount(ctx))
}

func TestReactorGetByBytes(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetReactorByBytes(ctx, []byte(item.Id))
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}

	// Test with nil bytes
	_, found := keeper.GetReactorByBytes(ctx, nil)
	require.False(t, found)
}

func TestReactorValidatorOperations(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create a test reactor
	reactor := types.Reactor{
		RawAddress: []byte("test-validator"),
	}
	reactor = keeper.AppendReactor(ctx, reactor)

	// Test getting reactor bytes from validator
	reactorBytes, found := keeper.GetReactorBytesFromValidator(ctx, []byte("test-validator"))
	require.True(t, found)
	require.Equal(t, reactor.Id, string(reactorBytes))

	// Test getting reactor bytes from non-existent validator
	_, found = keeper.GetReactorBytesFromValidator(ctx, []byte("non-existent"))
	require.False(t, found)

	// Test getting reactor bytes with nil validator address
	_, found = keeper.GetReactorBytesFromValidator(ctx, nil)
	require.False(t, found)
}
*/
