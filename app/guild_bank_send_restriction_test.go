package app_test

import (
	"bytes"
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"structs/app"
	"structs/app/upgrades"
	v0_21_0 "structs/app/upgrades/v0_21_0"
	keeperlib "structs/x/structs/keeper"
	structsmodule "structs/x/structs/module"
	structstypes "structs/x/structs/types"
)

func TestGuildBankSendRestrictionBlocksIBCAndUnregisteredRecipients(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	registered := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20))
	unregistered := sdk.AccAddress(bytes.Repeat([]byte{0x32}, 20))
	escrow := ibctransfertypes.GetEscrowAddress(ibctransfertypes.PortID, "channel-0")
	require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, registered.String(), 1))

	guildCoins := sdk.NewCoins(sdk.NewCoin("uguild.0-1", math.NewInt(30)))
	transfer := sdk.NewCoins(sdk.NewCoin("uguild.0-1", math.NewInt(10)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, guildCoins))

	err := bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, unregistered, transfer)
	var destinationErr *structstypes.GuildBankDestinationError
	require.ErrorAs(t, err, &destinationErr)

	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, structstypes.ModuleName, registered, transfer,
	))

	err = bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, escrow, transfer)
	require.ErrorAs(t, err, &destinationErr)

	alpha := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, alpha))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, escrow, alpha))
}

// TestGuildTokenMintSendAndAgreementStillWork is the blast-radius check for the
// holder gate: mint, player-to-player transfer (MsgSend and PlayerSend), and a
// guild-denominated agreement must still succeed between eligible accounts.
func TestGuildTokenMintSendAndAgreementStillWork(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	bankMs := bankkeeper.NewMsgServerImpl(bApp.BankKeeper)

	ownerAcc := sdk.AccAddress(bytes.Repeat([]byte{0x51}, 20))
	peerAcc := sdk.AccAddress(bytes.Repeat([]byte{0x52}, 20))
	strangerAcc := sdk.AccAddress(bytes.Repeat([]byte{0x53}, 20))
	owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, ownerAcc)
	peer := appendAppPlayer(t, bApp.StructsKeeper, ctx, peerAcc)

	guild := structstypes.CreateEmptyGuild()
	guild.Id = "0-910"
	guild.Index = 910
	guild.Owner = owner.Id
	guild.Creator = owner.Creator
	bApp.StructsKeeper.SetGuild(ctx, guild)
	owner.GuildId = guild.Id
	bApp.StructsKeeper.SetPlayer(ctx, owner)

	alpha := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(10_000)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, alpha))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, ownerAcc, alpha))

	_, err := bootstrapAppGuildTokens(t, bApp, ms, ctx, &structstypes.MsgGuildBankMint{
		Creator:     owner.Creator,
		AmountAlpha: 5_000,
		AmountToken: 5_000,
	})
	require.NoError(t, err, "treasury issuance and distribution to a registered holder must work")

	denom := "uguild." + guild.Id
	require.Equal(t, math.NewInt(5_000), bApp.BankKeeper.GetBalance(ctx, ownerAcc, denom).Amount)

	_, err = bankMs.Send(ctx, &banktypes.MsgSend{
		FromAddress: ownerAcc.String(),
		ToAddress:   peerAcc.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(100))),
	})
	require.NoError(t, err, "MsgSend of guild tokens between registered players must still work")
	require.Equal(t, math.NewInt(100), bApp.BankKeeper.GetBalance(ctx, peerAcc, denom).Amount)

	_, err = ms.PlayerSend(ctx, &structstypes.MsgPlayerSend{
		Creator:     peer.Creator,
		FromAddress: peer.Creator,
		ToAddress:   ownerAcc.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(40))),
	})
	require.NoError(t, err, "PlayerSend of guild tokens between registered players must still work")
	require.Equal(t, math.NewInt(4_940), bApp.BankKeeper.GetBalance(ctx, ownerAcc, denom).Amount)
	require.Equal(t, math.NewInt(60), bApp.BankKeeper.GetBalance(ctx, peerAcc, denom).Amount)

	_, err = bankMs.Send(ctx, &banktypes.MsgSend{
		FromAddress: ownerAcc.String(),
		ToAddress:   strangerAcc.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(1))),
	})
	var destinationErr *structstypes.GuildBankDestinationError
	require.ErrorAs(t, err, &destinationErr)

	_, err = ms.PlayerSend(ctx, &structstypes.MsgPlayerSend{
		Creator:     owner.Creator,
		FromAddress: owner.Creator,
		ToAddress:   strangerAcc.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(1))),
	})
	require.ErrorAs(t, err, &destinationErr)

	substation := structstypes.Substation{Id: keeperlib.GetObjectID(structstypes.ObjectType_substation, 910), Owner: owner.Id, Creator: owner.Creator}
	bApp.StructsKeeper.SetSubstation(ctx, substation)
	bApp.StructsKeeper.SetGridAttribute(
		ctx,
		keeperlib.GetGridAttributeIDByObjectId(structstypes.GridAttributeType_capacity, owner.Id),
		100000,
	)

	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	_, supplyErr := cc.NewAllocation(structstypes.AllocationType_dynamic, owner.Id, substation.Id, owner.Creator, owner.Id, 100000)
	require.NoError(t, supplyErr)
	providerCache := cc.NewProvider(structstypes.Provider{
		Owner:                       owner.Id,
		Creator:                     owner.Creator,
		SubstationId:                substation.Id,
		Rate:                        sdk.NewCoin(denom, math.NewInt(10)),
		AccessPolicy:                structstypes.ProviderAccessPolicy_openMarket,
		CapacityMinimum:             1,
		CapacityMaximum:             10000,
		DurationMinimum:             1,
		DurationMaximum:             1000000,
		ProviderCancellationPenalty: math.LegacyZeroDec(),
		ConsumerCancellationPenalty: math.LegacyZeroDec(),
	})
	cc.CommitAll()

	ownerVaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(owner.Id)
	require.NoError(t, err)
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, ownerAcc, ownerVaults[0], sdk.NewCoins(sdk.NewInt64Coin(denom, 2000))))

	_, err = ms.AgreementOpen(ctx, &structstypes.MsgAgreementOpen{
		Creator:    owner.Creator,
		ProviderId: providerCache.GetProviderId(),
		Capacity:   10,
		Duration:   20,
	})
	require.NoError(t, err, "opening a guild-denominated agreement must still reach the collateral pool")

	agreementIds := bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(ctx, providerCache.GetProviderId())
	require.Len(t, agreementIds, 1)

	collateralPool := keeperlib.GetProviderCollateralPoolLocation(providerCache.GetProviderId())
	require.True(t, bApp.BankKeeper.GetBalance(ctx, collateralPool, denom).Amount.IsPositive())
}

func setupAppGuildBank(t *testing.T, bApp *app.App, ctx sdk.Context, seed byte, guildIndex uint64) (structstypes.Player, structstypes.Guild, sdk.AccAddress) {
	t.Helper()

	ownerAcc := sdk.AccAddress(bytes.Repeat([]byte{seed}, 20))
	owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, ownerAcc)

	guild := structstypes.CreateEmptyGuild()
	guild.Id = keeperlib.GetObjectID(structstypes.ObjectType_guild, guildIndex)
	guild.Index = guildIndex
	guild.Owner = owner.Id
	guild.Creator = owner.Creator
	bApp.StructsKeeper.SetGuild(ctx, guild)
	owner.GuildId = guild.Id
	bApp.StructsKeeper.SetPlayer(ctx, owner)

	alpha := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(2_000)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, alpha))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, ownerAcc, alpha))

	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	_, err := bootstrapAppGuildTokens(t, bApp, ms, ctx, &structstypes.MsgGuildBankMint{
		Creator: owner.Creator, AmountAlpha: 1_000, AmountToken: 500,
	})
	require.NoError(t, err)

	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(owner.Id)
	require.NoError(t, err)
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, ownerAcc, vaults[0], bApp.BankKeeper.GetAllBalances(ctx, ownerAcc)))
	return owner, guild, vaults[0]
}

// TestGuildBankConvertAndConvertTokenWithSendRestriction runs alpha->token and
// token->token through the live bank restriction. Guild collateral pools are not
// on the uguild allowlist; these paths only send ualpha through them and mint
// tokens to the player's protocol vault.
func TestGuildBankConvertAndConvertTokenWithSendRestriction(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)

	ownerA, guildA, accA := setupAppGuildBank(t, bApp, ctx, 0x61, 911)
	_, guildB, _ := setupAppGuildBank(t, bApp, ctx, 0x62, 912)

	denomA := "uguild." + guildA.Id
	denomB := "uguild." + guildB.Id

	_, err := ms.GuildBankConvert(ctx, &structstypes.MsgGuildBankConvert{
		Creator: ownerA.Creator, GuildId: guildA.Id, AmountAlpha: 100, MinAmountToken: 1,
	})
	require.NoError(t, err, "converting ualpha into guild tokens must still credit the player")
	require.Equal(t, math.NewInt(550), bApp.BankKeeper.GetBalance(ctx, accA, denomA).Amount)

	// The first swap leg burns source tokens and releases collateral. A failed
	// second-leg price guard must roll all of that back with the SDK message cache.
	sourcePool := bApp.StructsKeeper.NewCurrentContext(ctx).GetGuild(guildA.Id).GetBankCollateralPool()
	targetPool := bApp.StructsKeeper.NewCurrentContext(ctx).GetGuild(guildB.Id).GetBankCollateralPool()
	beforeVault := bApp.BankKeeper.GetAllBalances(ctx, accA)
	beforeSourcePool := bApp.BankKeeper.GetAllBalances(ctx, sourcePool)
	beforeTargetPool := bApp.BankKeeper.GetAllBalances(ctx, targetPool)
	beforeSourceSupply := bApp.BankKeeper.GetSupply(ctx, denomA)
	beforeTargetSupply := bApp.BankKeeper.GetSupply(ctx, denomB)
	failedCtx, _ := ctx.CacheContext()
	_, err = ms.GuildBankConvertToken(failedCtx, &structstypes.MsgGuildBankConvertToken{
		Creator: ownerA.Creator, AmountToken: sdk.NewInt64Coin(denomA, 50),
		GuildId: guildB.Id, MinAmountToken: 1000000,
	})
	require.Error(t, err)
	require.Equal(t, int64(500), bApp.BankKeeper.GetBalance(failedCtx, accA, denomA).Amount.Int64(), "exercise a failure after the first leg burned tokens")
	require.Equal(t, beforeVault, bApp.BankKeeper.GetAllBalances(ctx, accA))
	require.Equal(t, beforeSourcePool, bApp.BankKeeper.GetAllBalances(ctx, sourcePool))
	require.Equal(t, beforeTargetPool, bApp.BankKeeper.GetAllBalances(ctx, targetPool))
	require.Equal(t, beforeSourceSupply, bApp.BankKeeper.GetSupply(ctx, denomA))
	require.Equal(t, beforeTargetSupply, bApp.BankKeeper.GetSupply(ctx, denomB))

	_, err = ms.GuildBankConvertToken(ctx, &structstypes.MsgGuildBankConvertToken{
		Creator:        ownerA.Creator,
		AmountToken:    sdk.NewCoin(denomA, math.NewInt(50)),
		GuildId:        guildB.Id,
		MinAmountToken: 1,
	})
	require.NoError(t, err, "swapping one guild token for another must still land on the player")
	require.Equal(t, math.NewInt(500), bApp.BankKeeper.GetBalance(ctx, accA, denomA).Amount)
	require.Equal(t, math.NewInt(50), bApp.BankKeeper.GetBalance(ctx, accA, denomB).Amount)
	primary, err := sdk.AccAddressFromBech32(ownerA.PrimaryAddress)
	require.NoError(t, err)
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, primary).IsZero(), "bank conversions must not route through a signing account")
}

func TestGuildDenominatedAgreementLifecycleWithSendRestriction(t *testing.T) {
	testGuildDenominatedAgreementLifecycle(t, false, false, 0)
}

func TestGuildOwnedAgreementLifecycleWithSendRestriction(t *testing.T) {
	testGuildDenominatedAgreementLifecycle(t, true, false, 0)
}

func TestGuildDenominatedAgreementBrownoutSettlesSharedPool(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, penaltyPercent := range []int64{0, 25, 100} {
			testGuildDenominatedAgreementLifecycle(t, guildOwned, true, penaltyPercent)
		}
	}
}

func testGuildDenominatedAgreementLifecycle(t *testing.T, guildOwned, brownout bool, penaltyPercent int64) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)

	consumerAcc := sdk.AccAddress(bytes.Repeat([]byte{0x41}, 20))
	ownerAcc := sdk.AccAddress(bytes.Repeat([]byte{0x42}, 20))
	consumer := appendAppPlayer(t, bApp.StructsKeeper, ctx, consumerAcc)
	owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, ownerAcc)

	substation := structstypes.Substation{
		Id:      keeperlib.GetObjectID(structstypes.ObjectType_substation, 900),
		Owner:   owner.Id,
		Creator: owner.Creator,
	}
	bApp.StructsKeeper.SetSubstation(ctx, substation)
	bApp.StructsKeeper.SetGridAttribute(
		ctx,
		keeperlib.GetGridAttributeIDByObjectId(structstypes.GridAttributeType_capacity, owner.Id),
		100000,
	)

	guildDenom := "uguild.0-900"
	guild := structstypes.CreateEmptyGuild()
	guild.Id = "0-900"
	guild.Index = 900
	guild.Owner = consumer.Id
	bApp.StructsKeeper.SetGuild(ctx, guild)

	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	feeding, supplyErr := cc.NewAllocation(structstypes.AllocationType_dynamic, owner.Id, substation.Id, owner.Creator, owner.Id, 100000)
	require.NoError(t, supplyErr)
	providerCache := cc.NewProvider(structstypes.Provider{
		Owner:                       owner.Id,
		Creator:                     owner.Creator,
		SubstationId:                substation.Id,
		Rate:                        sdk.NewCoin(guildDenom, math.NewInt(10)),
		AccessPolicy:                structstypes.ProviderAccessPolicy_openMarket,
		CapacityMinimum:             1,
		CapacityMaximum:             10000,
		DurationMinimum:             1,
		DurationMaximum:             1000000,
		ProviderCancellationPenalty: math.LegacyNewDec(penaltyPercent).QuoInt64(100),
		ConsumerCancellationPenalty: math.LegacyZeroDec(),
	})
	cc.CommitAll()

	funding := sdk.NewCoins(sdk.NewCoin(guildDenom, math.NewInt(100000)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, funding))
	consumerOwnerID := consumer.Id
	if guildOwned {
		consumerOwnerID = guild.Id
	}
	consumerVaults, vaultErr := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(consumerOwnerID)
	require.NoError(t, vaultErr)
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, consumerVaults[0], funding))

	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	_, err := ms.AgreementOpen(ctx, &structstypes.MsgAgreementOpen{
		Creator:    consumer.Creator,
		OwnerId:    consumerOwnerID,
		ProviderId: providerCache.GetProviderId(),
		Capacity:   100,
		Duration:   50,
	})
	require.NoError(t, err)

	agreementIds := bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(ctx, providerCache.GetProviderId())
	require.Len(t, agreementIds, 1)
	if brownout {
		// Two agreements share a collateral pool. Capacity loss must settle both
		// exactly once, checkpointing the served span before reducing load.
		peerAcc := sdk.AccAddress(bytes.Repeat([]byte{0x43}, 20))
		peer := appendAppPlayer(t, bApp.StructsKeeper, ctx, peerAcc)
		peerVaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(peer.Id)
		require.NoError(t, err)
		require.NoError(t, bApp.BankKeeper.SendCoins(ctx, consumerVaults[0], peerVaults[0],
			sdk.NewCoins(sdk.NewInt64Coin(guildDenom, 50000))))
		_, err = ms.AgreementOpen(ctx, &structstypes.MsgAgreementOpen{
			Creator: peer.Creator, OwnerId: peer.Id,
			ProviderId: providerCache.GetProviderId(), Capacity: 100, Duration: 50,
		})
		require.NoError(t, err)
		require.True(t, bApp.BankKeeper.GetBalance(ctx, consumerVaults[0], guildDenom).IsZero())
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 5)
		beforeEvents := len(ctx.EventManager().Events())
		cc = bApp.StructsKeeper.NewCurrentContext(ctx)
		supply, found := cc.GetAllocation(feeding.ID())
		require.True(t, found)
		require.NoError(t, supply.Destroy())
		cc.CommitAll()
		cc = bApp.StructsKeeper.NewCurrentContext(ctx)
		cc.GridCascade()
		cc.CommitAll()
		require.Empty(t, bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(ctx, providerCache.GetProviderId()))
		// 10,000 served units, with the provider penalty returned to consumers.
		expectedRefund := int64(90000) + 10000*penaltyPercent/100
		expectedEarnings := int64(100000) - expectedRefund
		require.Equal(t, expectedRefund/2, bApp.BankKeeper.GetBalance(ctx, consumerVaults[0], guildDenom).Amount.Int64())
		require.Equal(t, expectedRefund/2, bApp.BankKeeper.GetBalance(ctx, peerVaults[0], guildDenom).Amount.Int64())
		collateral := keeperlib.GetProviderCollateralPoolLocation(providerCache.GetProviderId())
		earnings := keeperlib.GetProviderEarningsPoolLocation(providerCache.GetProviderId())
		require.True(t, bApp.BankKeeper.GetBalance(ctx, collateral, guildDenom).IsZero())
		require.Equal(t, expectedEarnings, bApp.BankKeeper.GetBalance(ctx, earnings, guildDenom).Amount.Int64())
		require.True(t, bApp.BankKeeper.GetBalance(ctx, consumerAcc, guildDenom).IsZero())
		require.True(t, bApp.BankKeeper.GetBalance(ctx, peerAcc, guildDenom).IsZero())
		refundEvents := map[string]int{}
		for _, event := range ctx.EventManager().Events()[beforeEvents:] {
			if event.Type != "transfer" {
				continue
			}
			attributes := map[string]string{}
			for _, attribute := range event.Attributes {
				attributes[attribute.Key] = attribute.Value
			}
			if attributes["recipient"] == consumerVaults[0].String() || attributes["recipient"] == peerVaults[0].String() {
				require.Equal(t, collateral.String(), attributes["sender"])
				require.Equal(t, math.NewInt(expectedRefund/2).String()+guildDenom, attributes["amount"])
				refundEvents[attributes["recipient"]]++
			}
		}
		require.Equal(t, map[string]int{consumerVaults[0].String(): 1, peerVaults[0].String(): 1}, refundEvents,
			"each distinct consumer needs exactly one refund event")
		message, broken := keeperlib.ProviderCollateralSolvencyInvariant(bApp.StructsKeeper)(ctx)
		require.False(t, broken, message)
		bApp.StructsKeeper.AppendGridCascadeQueue(ctx, substation.Id)
		cc = bApp.StructsKeeper.NewCurrentContext(ctx)
		cc.GridCascade()
		cc.CommitAll()
		require.Equal(t, expectedRefund/2, bApp.BankKeeper.GetBalance(ctx, consumerVaults[0], guildDenom).Amount.Int64())
		require.Equal(t, expectedRefund/2, bApp.BankKeeper.GetBalance(ctx, peerVaults[0], guildDenom).Amount.Int64())
		require.Equal(t, expectedEarnings, bApp.BankKeeper.GetBalance(ctx, earnings, guildDenom).Amount.Int64())
		return
	}

	_, err = ms.AgreementDurationIncrease(ctx, &structstypes.MsgAgreementDurationIncrease{
		Creator:          consumer.Creator,
		AgreementId:      agreementIds[0],
		DurationIncrease: 10,
	})
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 5)
	_, err = ms.AgreementCapacityIncrease(ctx, &structstypes.MsgAgreementCapacityIncrease{
		Creator:          consumer.Creator,
		AgreementId:      agreementIds[0],
		CapacityIncrease: 50,
	})
	require.NoError(t, err)

	earningsPool := keeperlib.GetProviderEarningsPoolLocation(providerCache.GetProviderId())
	earned := bApp.BankKeeper.SpendableCoin(ctx, earningsPool, guildDenom).Amount
	require.True(t, earned.IsPositive())

	cc = bApp.StructsKeeper.NewCurrentContext(ctx)
	require.NoError(t, cc.GetGuild(guild.Id).BankConfiscateAndBurn(math.OneInt(), earningsPool.String()))
	cc.CommitAll()

	closing := bApp.StructsKeeper.NewCurrentContext(ctx).GetAgreement(agreementIds[0])
	require.Equal(t, consumerOwnerID, closing.GetOwnerId())
	expectedVault := bApp.BankKeeper.GetBalance(ctx, consumerVaults[0], guildDenom).Amount.Add(closing.GetRemainingCollateral())
	_, err = ms.AgreementClose(ctx, &structstypes.MsgAgreementClose{
		Creator:     consumer.Creator,
		AgreementId: agreementIds[0],
	})
	require.NoError(t, err)
	require.Empty(t, bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(ctx, providerCache.GetProviderId()))
	require.Equal(t, expectedVault, bApp.BankKeeper.GetBalance(ctx, consumerVaults[0], guildDenom).Amount)
	require.True(t, bApp.BankKeeper.GetBalance(ctx, consumerAcc, guildDenom).Amount.IsZero(), "consumer settlement stays in the asset owner's vault")

	invariant := keeperlib.ProviderCollateralSolvencyInvariant(bApp.StructsKeeper)
	message, broken := invariant(ctx)
	require.False(t, broken, message)

	_, err = ms.ProviderWithdrawBalance(ctx, &structstypes.MsgProviderWithdrawBalance{
		Creator:            owner.Creator,
		ProviderId:         providerCache.GetProviderId(),
		DestinationAddress: owner.Creator,
	})
	require.NoError(t, err)

	_, err = ms.ProviderDelete(ctx, &structstypes.MsgProviderDelete{
		Creator:    owner.Creator,
		ProviderId: providerCache.GetProviderId(),
	})
	require.NoError(t, err)

	oldCollateral := keeperlib.GetProviderCollateralPoolLocation(providerCache.GetProviderId())
	_, err = bApp.StructsKeeper.GuildBankDenomSendRestriction(
		ctx, consumerAcc, oldCollateral, sdk.NewCoins(sdk.NewCoin(guildDenom, math.OneInt())),
	)
	var destinationErr *structstypes.GuildBankDestinationError
	require.ErrorAs(t, err, &destinationErr)
}

func TestLegacyGuildEscrowMigrationProtectsExistingBacking(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	const channelId = "channel-7"
	bApp.IBCKeeper.ChannelKeeper.SetChannel(ctx, ibctransfertypes.PortID, channelId, channeltypes.Channel{
		State: channeltypes.OPEN,
	})
	escrow := ibctransfertypes.GetEscrowAddress(ibctransfertypes.PortID, channelId)

	guild := structstypes.CreateEmptyGuild()
	guild.Id = "0-901"
	guild.Index = 901
	bApp.StructsKeeper.SetGuild(ctx, guild)
	denom := "uguild." + guild.Id
	coin := sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(5)))

	fundEscrow := func() {
		bApp.BankKeeper.ClearSendRestriction()
		require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, coin))
		require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, escrow, coin))
		bApp.BankKeeper.AppendSendRestriction(bApp.StructsKeeper.GuildBankDenomSendRestriction)
	}
	confiscate := func() error {
		cc := bApp.StructsKeeper.NewCurrentContext(ctx)
		err := cc.GetGuild(guild.Id).BankConfiscateAndBurn(coin[0].Amount, escrow.String())
		if err == nil {
			cc.CommitAll()
		}
		return err
	}

	// This is the live pre-upgrade bug: escrow is just a bech32 account and the
	// privileged bank transfer accepts it as a source.
	fundEscrow()
	require.NoError(t, confiscate())

	fundEscrow()
	v0_21_0.MigrateProtectLegacyGuildEscrow(ctx, &upgrades.Keepers{
		StructsKeeper: bApp.StructsKeeper,
	})

	var protectedErr *structstypes.GuildBankConfiscationError
	require.ErrorAs(t, confiscate(), &protectedErr)
	require.Equal(t, "ibc_escrow_backing", protectedErr.Reason)

	_, err := bApp.StructsKeeper.GuildBankDenomSendRestriction(ctx, escrow, escrow, coin)
	var destinationErr *structstypes.GuildBankDestinationError
	require.ErrorAs(t, err, &destinationErr)

	exported := structsmodule.ExportGenesis(ctx, bApp.StructsKeeper)
	bApp.StructsKeeper.RemoveLegacyGuildBankEscrow(ctx, escrow.String())
	require.False(t, bApp.StructsKeeper.IsLegacyGuildBankEscrow(ctx, escrow.String()))
	structsmodule.InitGenesis(ctx, bApp.StructsKeeper, *exported)
	require.True(t, bApp.StructsKeeper.IsLegacyGuildBankEscrow(ctx, escrow.String()))
	require.ErrorAs(t, confiscate(), &protectedErr)
}

func appendAppPlayer(t *testing.T, k keeperlib.Keeper, ctx context.Context, address sdk.AccAddress) structstypes.Player {
	t.Helper()

	index := k.GetPlayerCount(ctx)
	player := structstypes.Player{
		Index:          index,
		Id:             keeperlib.GetObjectID(structstypes.ObjectType_player, index),
		Creator:        address.String(),
		PrimaryAddress: address.String(),
	}
	k.SetPlayer(ctx, player)
	k.SetPlayerCount(ctx, index+1)
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), index))
	k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), structstypes.PermAll)
	return player
}
