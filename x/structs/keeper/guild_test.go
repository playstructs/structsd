package keeper_test

/*
func createNGuild(keeper keeper.Keeper, ctx sdk.Context, n int) []types.Guild {
	items := make([]types.Guild, n)
	for i := range items {
		endpoint := "endpoint" + string(rune(i))
		substationId := "substation" + string(rune(i))
		reactor := types.Reactor{Id: "reactor" + string(rune(i))}
		player := types.Player{Id: "player" + string(rune(i)), Creator: "creator" + string(rune(i))}
		items[i] = keeper.AppendGuild(ctx, endpoint, substationId, reactor, player)
	}
	return items
}

func TestGuildGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuild(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetGuild(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestGuildRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuild(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveGuild(ctx, item.Id)
		_, found := keeper.GetGuild(ctx, item.Id)
		require.False(t, found)
	}
}

func TestGuildGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuild(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllGuild(ctx)),
	)
}

func TestGuildCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuild(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetGuildCount(ctx))
}

func createTestGuild(k keeper.Keeper, ctx sdk.Context, endpoint string, substationId string, reactor types.Reactor, player types.Player) types.Guild {
	return k.AppendGuild(ctx, endpoint, substationId, reactor, player)
}

func TestGuildBasicOperations(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	// Create test data
	endpoint := "test-endpoint"
	substationId := "substation1"
	reactor := types.Reactor{
		Id: "reactor1",
	}
	player := types.Player{
		Id:      "player1",
		Creator: "creator1",
	}

	// Test AppendGuild
	guild := createTestGuild(k, ctx, endpoint, substationId, reactor, player)
	require.NotEmpty(t, guild.Id)
	require.Equal(t, endpoint, guild.Endpoint)
	require.Equal(t, substationId, guild.EntrySubstationId)
	require.Equal(t, reactor.Id, guild.PrimaryReactorId)
	require.Equal(t, player.Id, guild.Owner)
	require.Equal(t, player.Creator, guild.Creator)

	// Test GetGuild
	retrievedGuild, found := k.GetGuild(ctx, guild.Id)
	require.True(t, found)
	require.Equal(t, guild.Id, retrievedGuild.Id)

	// Test SetGuild
	newEndpoint := "new-endpoint"
	retrievedGuild.Endpoint = newEndpoint
	k.SetGuild(ctx, retrievedGuild)

	updatedGuild, found := k.GetGuild(ctx, guild.Id)
	require.True(t, found)
	require.Equal(t, newEndpoint, updatedGuild.Endpoint)

	// Test RemoveGuild
	k.RemoveGuild(ctx, guild.Id)
	_, found = k.GetGuild(ctx, guild.Id)
	require.False(t, found)
}

func TestGuildCache(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	// Create test data
	endpoint := "test-endpoint"
	substationId := "substation1"
	reactor := types.Reactor{
		Id: "reactor1",
	}
	player := types.Player{
		Id:      "player1",
		Creator: "creator1",
	}

	// Create guild
	guild := createTestGuild(k, ctx, endpoint, substationId, reactor, player)

	// Test GuildCache
	cache := k.GetGuildCacheFromId(ctx, guild.Id)
	require.Equal(t, guild.Id, cache.GetGuildId())

	// Test loading guild data
	loadedGuild := cache.GetGuild()
	require.Equal(t, guild.Id, loadedGuild.Id)
	require.Equal(t, endpoint, loadedGuild.Endpoint)

	// Test owner loading
	owner, err := k.GetPlayerCacheFromId(ctx, player.Id)
	require.NoError(t, err)
	require.NotNil(t, owner)
	require.Equal(t, player.Id, owner.GetPlayerId())
}

func TestGuildBanking(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	// Create test data
	endpoint := "test-endpoint"
	substationId := "substation1"
	reactor := types.Reactor{
		Id: "reactor1",
	}
	player := types.Player{
		Id:      "player1",
		Creator: "creator1",
	}

	// Create guild
	guild := createTestGuild(k, ctx, endpoint, substationId, reactor, player)
	cache := k.GetGuildCacheFromId(ctx, guild.Id)

	// Test minting
	amountAlpha := math.NewInt(1000)
	amountToken := math.NewInt(100)

	// First ensure player has enough alpha
	playerAcc, _ := sdk.AccAddressFromBech32(player.Creator)
	alphaCoin := sdk.NewCoin("ualpha", amountAlpha)
	k.BankKeeper().MintCoins(ctx, types.ModuleName, sdk.NewCoins(alphaCoin))
	k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, sdk.NewCoins(alphaCoin))

	// Test minting
	err := cache.BankMint(amountAlpha, amountToken, cache.GetOwner())
	require.NoError(t, err)

	// Verify token balance
	tokenBalance := k.BankKeeper().SpendableCoin(ctx, playerAcc, cache.GetBankDenom())
	require.Equal(t, amountToken, tokenBalance.Amount)

	// Test redeeming
	err = cache.BankRedeem(amountToken, cache.GetOwner())
	require.NoError(t, err)

	// Verify alpha balance returned
	alphaBalance := k.BankKeeper().SpendableCoin(ctx, playerAcc, "ualpha")
	require.Equal(t, amountAlpha, alphaBalance.Amount)
}

func TestGuildPermissions(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	// Create test data
	endpoint := "test-endpoint"
	substationId := "substation1"
	reactor := types.Reactor{
		Id: "reactor1",
	}
	player := types.Player{
		Id:      "player1",
		Creator: "creator1",
	}

	// Create guild
	guild := createTestGuild(k, ctx, endpoint, substationId, reactor, player)
	cache := k.GetGuildCacheFromId(ctx, guild.Id)

	// Test owner permissions
	err := cache.CanUpdate(cache.GetOwner())
	require.NoError(t, err)

	err = cache.CanDelete(cache.GetOwner())
	require.NoError(t, err)

	err = cache.CanAdministrateBank(cache.GetOwner())
	require.NoError(t, err)

	// Test non-owner permissions (should fail)
	otherPlayer := types.Player{
		Id:      "player2",
		Creator: "creator2",
	}
	otherCache, err := k.GetPlayerCacheFromId(ctx, otherPlayer.Id)
	require.NoError(t, err)

	err = cache.CanUpdate(&otherCache)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no permissions")
}
*/
