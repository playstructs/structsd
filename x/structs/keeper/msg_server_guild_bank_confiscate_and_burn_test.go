package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildBankConfiscateAndBurn(t *testing.T) {
	for _, scenario := range []string{"vault", "key_address", "unregistered_signer", "no_burn_permission", "unprivileged_player", "insufficient_tokens", "invalid_address"} {
		t.Run(scenario, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			primary := sdk.AccAddress("burn-owner-primary01")
			player := testAppendPlayer(k, ctx, types.Player{Creator: primary.String(), PrimaryAddress: primary.String()})
			holderKey := sdk.AccAddress("burn-holder-primary1")
			holder := testAppendPlayer(k, ctx, types.Player{Creator: holderKey.String(), PrimaryAddress: holderKey.String()})
			reactor := k.AppendReactor(ctx, types.Reactor{RawAddress: sdk.ValAddress(primary.Bytes()).Bytes()})
			guild := k.AppendGuild(ctx, "test-endpoint", "", reactor, player, "")
			player.GuildId = guild.Id
			k.SetPlayer(ctx, player)
			cc := k.NewCurrentContext(ctx)
			vaults, err := cc.EnsureVaults(guild.Id, holder.Id)
			require.NoError(t, err)
			cc.CommitAll()
			treasury, err := types.VaultAddress(guild.Id)
			require.NoError(t, err)
			holderVault, err := types.VaultAddress(holder.Id)
			require.NoError(t, err)
			require.Len(t, vaults, 2)
			fundAlpha(t, k, ctx, treasury, 1000)
			fundAlpha(t, k, ctx, holderVault, 333)
			fundAlpha(t, k, ctx, holderKey, 777)
			_, err = ms.GuildBankMint(ctx, &types.MsgGuildBankMint{Creator: player.Creator, AmountAlpha: 100, AmountToken: 10})
			require.NoError(t, err)
			denom := "uguild." + guild.Id
			require.NoError(t, k.BankKeeper().SendCoins(ctx, treasury, holderVault, sdk.NewCoins(sdk.NewInt64Coin(denom, 7))))
			require.NoError(t, k.BankKeeper().SendCoins(ctx, treasury, holderKey, sdk.NewCoins(sdk.NewInt64Coin(denom, 2))))
			msg := &types.MsgGuildBankConfiscateAndBurn{Creator: player.Creator, GuildId: guild.Id, Address: holderVault.String(), AmountToken: 5}
			switch scenario {
			case "key_address":
				msg.Address, msg.AmountToken = holderKey.String(), 1
			case "unregistered_signer":
				msg.Creator = sdk.AccAddress("burn-unknown-signer1").String()
			case "no_burn_permission":
				testPermissionRemove(k, ctx, keeperlib.GetAddressPermissionIDBytes(player.Creator), types.PermGuildTokenBurn)
			case "unprivileged_player":
				msg.Creator = holder.Creator
			case "insufficient_tokens":
				msg.AmountToken = 8
			case "invalid_address":
				msg.Address = "invalid"
			}
			indexBefore := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
			collateralBefore := collateralBalance(k, ctx, guild.Id)
			supplyBefore := k.BankKeeper().GetSupply(ctx, denom).Amount
			_, err = ms.GuildBankConfiscateAndBurn(ctx, msg)
			wantVault, wantKey, burned := int64(7), int64(2), int64(0)
			switch scenario {
			case "vault":
				require.NoError(t, err)
				wantVault, burned = 2, 5
			case "key_address":
				require.NoError(t, err)
				wantKey, burned = 1, 1
			default:
				require.Error(t, err)
				if scenario == "no_burn_permission" {
					requireDeniedOnSignerBit(t, err, msg.Creator, types.PermGuildTokenBurn)
				}
			}
			require.Equal(t, wantVault, k.BankKeeper().SpendableCoin(ctx, holderVault, denom).Amount.Int64())
			require.Equal(t, wantKey, k.BankKeeper().SpendableCoin(ctx, holderKey, denom).Amount.Int64())
			require.Equal(t, int64(1), k.BankKeeper().SpendableCoin(ctx, treasury, denom).Amount.Int64())
			require.True(t, supplyBefore.SubRaw(burned).Equal(k.BankKeeper().GetSupply(ctx, denom).Amount))
			require.True(t, collateralBefore.Equal(collateralBalance(k, ctx, guild.Id)), "confiscation does not redeem collateral")
			require.Equal(t, int64(333), k.BankKeeper().SpendableCoin(ctx, holderVault, "ualpha").Amount.Int64())
			require.Equal(t, int64(777), k.BankKeeper().SpendableCoin(ctx, holderKey, "ualpha").Amount.Int64())
			require.Equal(t, indexBefore, k.GetPlayerIndexFromAddress(ctx, msg.Creator), "confiscation cannot implicitly register a caller")
		})
	}
}

// A zero confiscation amount must be rejected up front, matching the sibling
// bank operations, rather than committing and emitting a zero-value
// EventGuildBankConfiscateAndBurn that indexers ingest as a real confiscation.
func TestMsgGuildBankConfiscateAndBurn_RejectsZeroAmount(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	})

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	reactor := k.AppendReactor(ctx, types.Reactor{RawAddress: validatorAddress.Bytes()})
	guild := k.AppendGuild(ctx, "test-endpoint", "", reactor, player, "")
	player.GuildId = guild.Id
	k.SetPlayer(ctx, player)

	_, err := ms.GuildBankConfiscateAndBurn(wctx, &types.MsgGuildBankConfiscateAndBurn{
		Creator:     player.Creator,
		AmountToken: 0,
		Address:     player.Creator,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must_be_positive")
}
