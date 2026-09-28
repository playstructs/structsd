package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructGeneratorInfuse(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	player := types.Player{
		Creator:        sdk.AccAddress("generator-primary-01").String(),
		PrimaryAddress: sdk.AccAddress("generator-primary-01").String(),
	}
	player = testAppendPlayer(k, ctx, player)

	planet := testAppendPlanet(k, ctx, types.Planet{Creator: player.Creator, Owner: player.Id})

	structType := types.StructType{
		Id:              1,
		Type:            types.CommandStruct,
		Category:        types.ObjectType_player,
		PowerGeneration: 1,
	}
	k.SetStructType(ctx, structType)

	structObj := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   planet.Id,
		LocationType: types.ObjectType_planet,
	}
	structObj = testAppendStruct(k, ctx, structObj)

	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))

	playerAcc, err := types.VaultAddress(player.Id)
	require.NoError(t, err)
	coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1000)))
	k.BankKeeper().MintCoins(ctx, types.ModuleName, coins)
	k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, coins)

	primaryAcc, err := sdk.AccAddressFromBech32(player.PrimaryAddress)
	require.NoError(t, err)
	primaryFunds := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 7000))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, primaryFunds))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, primaryAcc, primaryFunds))

	t.Run("valid generator infuse", func(t *testing.T) {
		resp, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     structObj.Id,
			InfuseAmount: "1000ualpha",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, k.BankKeeper().SpendableCoin(ctx, playerAcc, "ualpha").Amount.IsZero())
		require.Equal(t, math.NewInt(7000), k.BankKeeper().SpendableCoin(ctx, primaryAcc, "ualpha").Amount)
		infusion, found := k.GetInfusion(ctx, structObj.Id, playerAcc.String())
		require.True(t, found)
		require.Equal(t, uint64(1000), infusion.Fuel)
		require.Equal(t, player.Id, infusion.AssetOwnerID())
		_, found = k.GetInfusion(ctx, structObj.Id, primaryAcc.String())
		require.False(t, found)
		owner, found := k.GetVaultOwner(ctx, playerAcc)
		require.True(t, found)
		require.Equal(t, player.Id, owner)
		count := 0
		for _, event := range wctx.EventManager().ABCIEvents() {
			if event.Type != "structs.structs.EventAlphaInfuse" { continue }
			parsed, parseErr := sdk.ParseTypedEvent(event)
			require.NoError(t, parseErr)
			detail := parsed.(*types.EventAlphaInfuse).EventAlphaInfuseDetail
			require.Equal(t, playerAcc.String(), detail.VaultAddress)
			require.Equal(t, player.PrimaryAddress, detail.PrimaryAddress)
			require.Equal(t, player.Id, detail.PlayerId)
			require.Equal(t, uint64(1000), detail.Amount)
			count++
		}
		require.Equal(t, 1, count)
	})

	t.Run("empty vault does not fall back to primary funds", func(t *testing.T) {
		_, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator: player.Creator, StructId: structObj.Id, InfuseAmount: "1ualpha",
		})
		require.Error(t, err)
		require.Equal(t, math.NewInt(7000), k.BankKeeper().SpendableCoin(ctx, primaryAcc, "ualpha").Amount)
		infusion, found := k.GetInfusion(ctx, structObj.Id, playerAcc.String())
		require.True(t, found)
		require.Equal(t, uint64(1000), infusion.Fuel)
	})

	t.Run("struct offline", func(t *testing.T) {
		testSetStructAttributeFlagRemove(k, ctx, statusAttrId, uint64(types.StructStateOnline))

		resp, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     structObj.Id,
			InfuseAmount: "1000ualpha",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "is offline")
		_ = resp

		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))
	})

	t.Run("no power generation", func(t *testing.T) {
		noGenType := types.StructType{
			Id:              2,
			Type:            types.CommandStruct,
			Category:        types.ObjectType_player,
			PowerGeneration: types.TechPowerGeneration_noPowerGeneration,
		}
		k.SetStructType(ctx, noGenType)
		structObj.Type = noGenType.Id
		k.SetStruct(ctx, structObj)

		resp, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     structObj.Id,
			InfuseAmount: "1000ualpha",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "has no generation system")
		_ = resp

		structObj.Type = structType.Id
		k.SetStruct(ctx, structObj)
	})

	t.Run("no permissions", func(t *testing.T) {
		resp, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      "cosmos1noperms",
			StructId:     structObj.Id,
			InfuseAmount: "1000ualpha",
		})
		require.Error(t, err)
		_ = resp
	})
}

/* TestStructGeneratorInfuseBurnsOnlyTheFuelItChecked is the regression on a
 * validated first coin standing in for the whole collection.
 *
 * msg.InfuseAmount is a free-form string and ParseCoinsNormalized accepts a
 * comma-separated list, sorting it by denom. Checking infusionAmount[0] and then
 * moving the whole slice therefore validated one coin and burned all of them:
 * "1ualpha,1000000uguild.0-1" sorts ualpha first, passes the denom check, and
 * destroys the guild tokens - which may share the vault
 * account this handler spends from. Only the first coin was ever credited as
 * fuel, so the rest went for nothing.
 *
 * The authorization boundary is the part that makes it more than an accounting
 * bug: this handler asks only for PermTokenInfuse, so a delegate scoped to
 * infusion alone could destroy asset classes PermGuildTokenBurn exists to
 * protect.
 */
func TestStructGeneratorInfuseBurnsOnlyTheFuelItChecked(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        "cosmos1fuelchecked",
		PrimaryAddress: "cosmos1fuelchecked",
	})

	planet := testAppendPlanet(k, ctx, types.Planet{Creator: player.Creator, Owner: player.Id})

	k.SetStructType(ctx, types.StructType{
		Id:              7,
		Type:            types.CommandStruct,
		Category:        types.ObjectType_player,
		PowerGeneration: 1,
	})

	generator := testAppendStruct(k, ctx, types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         7,
		LocationId:   planet.Id,
		LocationType: types.ObjectType_planet,
	})
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, generator.Id)
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))

	playerAcc, err := types.VaultAddress(player.Id)
	require.NoError(t, err)

	// A guild token in the same account. "ualpha" sorts before "uguild.", which
	// is what put a valid denom at index 0 in front of one that is not.
	const guildDenom = "uguild.0-1"
	fund := func(denom string, amount int64) {
		coins := sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(amount)))
		require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
		require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, coins))
	}
	fund("ualpha", 5000)
	fund(guildDenom, 1_000_000)

	guildBefore := k.BankKeeper().SpendableCoin(ctx, playerAcc, guildDenom).Amount

	t.Run("a second denom is refused outright", func(t *testing.T) {
		_, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     generator.Id,
			InfuseAmount: "1ualpha,1000000" + guildDenom,
		})
		require.Error(t, err, "a coin the handler never validated must not be moved")

		require.Equal(t, guildBefore, k.BankKeeper().SpendableCoin(ctx, playerAcc, guildDenom).Amount,
			"the guild tokens must be untouched")
	})

	t.Run("a non-fuel denom on its own is still refused", func(t *testing.T) {
		_, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     generator.Id,
			InfuseAmount: "1000" + guildDenom,
		})
		require.Error(t, err)
		require.Equal(t, guildBefore, k.BankKeeper().SpendableCoin(ctx, playerAcc, guildDenom).Amount)
	})

	// The controls: a single fuel coin still works, in both denominations.
	t.Run("a single ualpha coin still infuses", func(t *testing.T) {
		alphaBefore := k.BankKeeper().SpendableCoin(ctx, playerAcc, "ualpha").Amount

		_, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     generator.Id,
			InfuseAmount: "1000ualpha",
		})
		require.NoError(t, err)

		require.Equal(t, alphaBefore.SubRaw(1000), k.BankKeeper().SpendableCoin(ctx, playerAcc, "ualpha").Amount)
		require.Equal(t, guildBefore, k.BankKeeper().SpendableCoin(ctx, playerAcc, guildDenom).Amount)
	})

	t.Run("alpha is still converted to ualpha", func(t *testing.T) {
		fund("ualpha", 2_000_000)
		alphaBefore := k.BankKeeper().SpendableCoin(ctx, playerAcc, "ualpha").Amount

		_, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     generator.Id,
			InfuseAmount: "1alpha",
		})
		require.NoError(t, err)

		require.Equal(t, alphaBefore.SubRaw(1_000_000), k.BankKeeper().SpendableCoin(ctx, playerAcc, "ualpha").Amount,
			"one alpha spends a million ualpha")
	})
}
