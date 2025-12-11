package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

func createNGuild(keeper keeper.Keeper, ctx sdk.Context, n int) []types.Guild {
	items := make([]types.Guild, n)
	for i := range items {
		endpoint := "endpoint" + string(rune(i))
		substationId := "substation" + string(rune(i))
		reactor := types.Reactor{Id: "reactor" + string(rune(i))}
		// Create a real player first
		player := types.Player{
			Creator:        "creator" + string(rune(i)),
			PrimaryAddress: "creator" + string(rune(i)),
		}
		player = keeper.AppendPlayer(ctx, player)
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
		require.Equal(t, item.Id, got.Id)
		require.Equal(t, item.Endpoint, got.Endpoint)
		require.Equal(t, item.EntrySubstationId, got.EntrySubstationId)
		require.Equal(t, item.PrimaryReactorId, got.PrimaryReactorId)
		require.Equal(t, item.Owner, got.Owner)
		require.Equal(t, item.Creator, got.Creator)
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
	got := keeper.GetAllGuild(ctx)
	require.GreaterOrEqual(t, len(got), len(items), "Should have at least the created items")
	// Verify all created items are in the result
	for _, item := range items {
		found := false
		for _, guild := range got {
			if guild.Id == item.Id {
				found = true
				require.Equal(t, item.Endpoint, guild.Endpoint)
				require.Equal(t, item.EntrySubstationId, guild.EntrySubstationId)
				require.Equal(t, item.PrimaryReactorId, guild.PrimaryReactorId)
				require.Equal(t, item.Owner, guild.Owner)
				require.Equal(t, item.Creator, guild.Creator)
				break
			}
		}
		require.True(t, found, "Guild %s should be in GetAllGuild result", item.Id)
	}
}

func TestGuildCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	initialCount := keeper.GetGuildCount(ctx)
	items := createNGuild(keeper, ctx, 10)
	expectedCount := initialCount + uint64(len(items))
	actualCount := keeper.GetGuildCount(ctx)
	require.Equal(t, expectedCount, actualCount)
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
	// Create a real player first
	player := types.Player{
		Creator:        "creator1",
		PrimaryAddress: "creator1",
	}
	player = k.AppendPlayer(ctx, player)

	// Create guild
	guild := createTestGuild(k, ctx, endpoint, substationId, reactor, player)
	cache := k.GetGuildCacheFromId(ctx, guild.Id)

	// Get owner player cache with proper address set
	ownerCache, err := k.GetPlayerCacheFromAddress(ctx, player.PrimaryAddress)
	require.NoError(t, err)

	// Ensure the owner has address permissions (set when player is created)
	// The owner should have PermissionAll on their address

	// Test owner permissions
	err = cache.CanUpdate(&ownerCache)
	require.NoError(t, err)

	err = cache.CanDelete(&ownerCache)
	require.NoError(t, err)

	err = cache.CanAdministrateBank(&ownerCache)
	require.NoError(t, err)

	// Test non-owner permissions (should fail)
	// Create a real player first
	otherPlayer := types.Player{
		Creator:        "creator2",
		PrimaryAddress: "creator2",
	}
	otherPlayer = k.AppendPlayer(ctx, otherPlayer)
	otherCache, err := k.GetPlayerCacheFromAddress(ctx, otherPlayer.PrimaryAddress)
	require.NoError(t, err)

	err = cache.CanUpdate(&otherCache)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no")
}
