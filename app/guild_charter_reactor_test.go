package app_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Paid founding requires a real, unjailed validator but no ownership entitlement.
// Selecting a reactor is a guild preference and conveys no reactor authority.
func TestPaidGuildCreationWithUncontrolledReactor(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	founder := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x71}, 20)))
	operator := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x72}, 20)))
    validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
    require.NoError(t, err)
    validatorAddress, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
    require.NoError(t, err)
	reactor := bApp.StructsKeeper.AppendReactor(ctx, types.Reactor{Owner: operator.Id, Validator: validatorAddress.String(), RawAddress: validatorAddress.Bytes(), GuildId: "0-999"})
	storedReactor, existed := bApp.StructsKeeper.GetReactor(ctx, reactor.Id)
	require.True(t, existed)
	addresses, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(founder.Id)
	require.NoError(t, err)
	funds := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 2000000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, types.ModuleName, funds))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addresses[0], funds))
	beforeSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	response, err := ms.GuildCreate(ctx, &types.MsgGuildCreate{Creator: founder.Creator, ReactorId: reactor.Id})
	require.NoError(t, err)
	guild, found := bApp.StructsKeeper.GetGuild(ctx, response.GuildId)
	require.True(t, found)
	require.Equal(t, founder.Id, guild.Owner)
	require.Equal(t, reactor.Id, guild.PrimaryReactorId)
	address, err := types.VaultAddress(guild.Id)
	require.NoError(t, err)
	require.Equal(t, int64(1000000), bApp.BankKeeper.GetBalance(ctx, address, "uguild."+guild.Id).Amount.Int64())
	require.Equal(t, int64(1000000), beforeSupply.Sub(bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount).Int64())
	require.True(t, bApp.BankKeeper.GetBalance(ctx, addresses[0], "ualpha").IsZero())
	pool := bApp.StructsKeeper.NewCurrentContext(ctx).GetGuild(guild.Id).GetBankCollateralPool()
	require.Equal(t, int64(1000000), bApp.BankKeeper.GetBalance(ctx, pool, "ualpha").Amount.Int64())
	after, found := bApp.StructsKeeper.GetReactor(ctx, reactor.Id)
	require.True(t, found)
	require.Equal(t, storedReactor, after)
	player, found := bApp.StructsKeeper.GetPlayer(ctx, founder.Id)
	require.True(t, found)
	require.Equal(t, guild.Id, player.GuildId)
	moduleAddress := bApp.AccountKeeper.GetModuleAddress(types.ModuleName).String()
	burns, mints, vaultEvents, bankAssociations := 0, 0, 0, 0
	vaultEventIndex, creditEventIndex := -1, -1
	var transfers [][3]string
	for index, event := range ctx.EventManager().Events() {
		switch event.Type {
		case "structs.structs.EventVaultAddress":
			decoded, decodeErr := sdk.ParseTypedEvent(abci.Event(event))
			require.NoError(t, decodeErr)
			detail := decoded.(*types.EventVaultAddress).EventVaultAddressDetail
			require.Equal(t, guild.Id, detail.OwnerId)
			require.Equal(t, address.String(), detail.VaultAddress)
			require.True(t, detail.AccountCreated)
			require.Equal(t, bApp.AccountKeeper.GetAccount(ctx, address).GetAccountNumber(), detail.AccountNumber)
			vaultEvents++
			vaultEventIndex = index
		case "structs.structs.EventGuildBankAddress":
			decoded, decodeErr := sdk.ParseTypedEvent(abci.Event(event))
			require.NoError(t, decodeErr)
			detail := decoded.(*types.EventGuildBankAddress).EventGuildBankAddressDetail
			require.Equal(t, guild.Id, detail.GuildId)
			require.Equal(t, pool.String(), detail.BankCollateralPool)
			require.Equal(t, types.ModuleName, detail.BankTokenPool)
			bankAssociations++
		case banktypes.EventTypeTransfer:
			attributes := make(map[string]string)
			for _, attribute := range event.Attributes {
				require.NotContains(t, attributes, attribute.Key)
				attributes[attribute.Key] = attribute.Value
			}
			transfers = append(transfers, [3]string{attributes["sender"], attributes["recipient"], attributes["amount"]})
			if attributes["recipient"] == address.String() {
				creditEventIndex = index
			}
		case banktypes.EventTypeCoinBurn:
			burns++
			require.Equal(t, sdk.NewEvent(banktypes.EventTypeCoinBurn,
				sdk.NewAttribute(banktypes.AttributeKeyBurner, moduleAddress),
				sdk.NewAttribute(sdk.AttributeKeyAmount, "1000000ualpha")), event)
		case banktypes.EventTypeCoinMint:
			mints++
			require.Equal(t, sdk.NewEvent(banktypes.EventTypeCoinMint,
				sdk.NewAttribute(banktypes.AttributeKeyMinter, moduleAddress),
				sdk.NewAttribute(sdk.AttributeKeyAmount, "1000000uguild."+guild.Id)), event)
		}
	}
	require.Equal(t, 1, burns)
	require.Equal(t, 1, mints)
	require.Equal(t, 1, vaultEvents)
	require.Equal(t, 1, bankAssociations)
	require.ElementsMatch(t, [][3]string{
		{addresses[0].String(), moduleAddress, "1000000ualpha"},
		{addresses[0].String(), pool.String(), "1000000ualpha"},
		{moduleAddress, address.String(), "1000000uguild." + guild.Id},
	}, transfers)
	require.Greater(t, creditEventIndex, vaultEventIndex, "associate the vault before indexing its credit")
	vaultOwner, associated := bApp.StructsKeeper.GetVaultOwner(ctx, address)
	require.True(t, associated)
	require.Equal(t, guild.Id, vaultOwner)
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, address.String()), "vaults are not signing keys")
}

// A failed final credit occurs after bank supply and auth accounts have changed
// in the transaction cache. None of those changes may escape a discarded cache.
func TestPaidGuildCreationFinalCreditFailureRollsBack(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	founder := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x73}, 20)))
    validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
    require.NoError(t, err)
    validatorAddress, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
    require.NoError(t, err)
	reactor := bApp.StructsKeeper.AppendReactor(ctx, types.Reactor{Owner: founder.Id, Validator: validatorAddress.String(), RawAddress: validatorAddress.Bytes()})
	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(founder.Id)
	require.NoError(t, err)
	funds := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 2000000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, types.ModuleName, funds))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, vaults[0], funds))
	countBefore := bApp.StructsKeeper.GetGuildCount(ctx)
	guildID := keeperlib.GetObjectID(types.ObjectType_guild, countBefore)
	treasury, err := types.VaultAddress(guildID)
	require.NoError(t, err)
	denom := "uguild." + guildID
	beforeSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount
	permissionsBefore := bApp.StructsKeeper.GetAllPermissionExport(ctx)
	playerBefore, found := bApp.StructsKeeper.GetPlayer(ctx, founder.Id)
	require.True(t, found)
	rejectCredit, refusals := true, 0
	bApp.BankKeeper.AppendSendRestriction(func(_ context.Context, from, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
		if rejectCredit && to.Equals(treasury) && coins.AmountOf(denom).IsPositive() {
			refusals++
			return to, errors.New("injected guild token credit refusal")
		}
		return to, nil
	})
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	failedCtx, _ := ctx.CacheContext()
	msg := &types.MsgGuildCreate{Creator: founder.Creator, ReactorId: reactor.Id}
	_, err = ms.GuildCreate(failedCtx, msg)
	require.ErrorContains(t, err, "injected guild token credit refusal")
	require.Equal(t, 1, refusals)
	pool := bApp.StructsKeeper.NewCurrentContext(failedCtx).GetGuild(guildID).GetBankCollateralPool()
	require.Equal(t, int64(1000000), beforeSupply.Sub(bApp.BankKeeper.GetSupply(failedCtx, "ualpha").Amount).Int64())
	require.Equal(t, int64(1000000), bApp.BankKeeper.GetSupply(failedCtx, denom).Amount.Int64())
	require.Equal(t, int64(1000000), bApp.BankKeeper.GetBalance(failedCtx, pool, "ualpha").Amount.Int64())
	failedVaultAccount := bApp.AccountKeeper.GetAccount(failedCtx, treasury)
	require.NotNil(t, failedVaultAccount)
	failedPoolAccount := bApp.AccountKeeper.GetAccount(failedCtx, pool)
	require.NotNil(t, failedPoolAccount)
	require.NotEmpty(t, failedCtx.EventManager().Events())

	_, found = bApp.StructsKeeper.GetGuild(ctx, guildID)
	require.False(t, found)
	require.Equal(t, countBefore, bApp.StructsKeeper.GetGuildCount(ctx))
	require.Equal(t, beforeSupply, bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount)
	require.True(t, bApp.BankKeeper.GetSupply(ctx, denom).IsZero())
	require.Equal(t, funds[0], bApp.BankKeeper.GetBalance(ctx, vaults[0], "ualpha"))
	require.True(t, bApp.BankKeeper.GetBalance(ctx, pool, "ualpha").IsZero())
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, treasury))
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, pool))
	_, associated := bApp.StructsKeeper.GetVaultOwner(ctx, treasury)
	require.False(t, associated)
	_, metadataExists := bApp.BankKeeper.GetDenomMetaData(ctx, denom)
	require.False(t, metadataExists)
	require.Equal(t, permissionsBefore, bApp.StructsKeeper.GetAllPermissionExport(ctx))
	playerAfter, found := bApp.StructsKeeper.GetPlayer(ctx, founder.Id)
	require.True(t, found)
	require.Equal(t, playerBefore, playerAfter)
	require.Empty(t, ctx.EventManager().Events())

	rejectCredit = false
	response, err := ms.GuildCreate(ctx, msg)
	require.NoError(t, err)
	require.Equal(t, guildID, response.GuildId)
	require.Equal(t, failedVaultAccount.GetAccountNumber(), bApp.AccountKeeper.GetAccount(ctx, treasury).GetAccountNumber())
	require.Equal(t, failedPoolAccount.GetAccountNumber(), bApp.AccountKeeper.GetAccount(ctx, pool).GetAccountNumber())
	require.Equal(t, int64(1000000), bApp.BankKeeper.GetBalance(ctx, treasury, denom).Amount.Int64())
	require.Equal(t, int64(1000000), beforeSupply.Sub(bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount).Int64())
}
