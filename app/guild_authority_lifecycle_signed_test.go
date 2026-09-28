package app_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Exercise actual signed transitions rather than seeding the final departed or
// demoted state. A direct self-grant made while authorized is independent of the
// rank/owner authority that allowed it, until explicitly revoked.
func TestSignedGuildAuthorityLifecyclePreservesIndependentChildGrant(t *testing.T) {
	for _, transition := range []string{"departure", "kick", "demotion", "rank_revoke", "ownership_transfer"} {
		t.Run(transition, func(t *testing.T) {
			bApp, ctx := setupReactorAppGenesis(t, 1, true)
			k := bApp.StructsKeeper
			actorKey, ownerKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
			actorAddress, ownerAddress := sdk.AccAddress(actorKey.PubKey().Address()), sdk.AccAddress(ownerKey.PubKey().Address())
			actor := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
			owner := appendAppPlayer(t, k, ctx, ownerAddress)
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, actorAddress.String(), actor.Index))
			bits := types.PermUpdate | types.PermGuildMembership | types.PermTokenTransfer
			guild, destination := types.CreateEmptyGuild(), types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner = "0-992", 992, owner.Id
			guild.EntryRank, guild.RankPurchaseMinimumRank, guild.RankPurchaseBasePrice = 5, 1, 1
			destination.Id, destination.Index, destination.Owner = "0-993", 993, owner.Id
			destination.EntryRank, destination.RankPurchaseMinimumRank, destination.JoinByRankPurchaseEnabled = 5, 1, true
			actor.GuildId, actor.GuildRank = guild.Id, 5
			if transition == "ownership_transfer" {
				guild.Owner = actor.Id
				actor.GuildId, actor.GuildRank = "", 0
				bits |= types.PermAdmin
			}
			k.SetGuild(ctx, guild)
			k.SetGuild(ctx, destination)
			k.SetPlayer(ctx, actor)
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(actorAddress.String()), bits)
			direct := types.Substation{Id: "4-992", Owner: guild.Id, Name: "Direct Grant"}
			inherited := types.Substation{Id: "4-993", Owner: guild.Id, Name: "Inherited Grant"}
			for _, station := range []types.Substation{direct, inherited} {
				k.SetSubstation(ctx, station)
				if transition != "ownership_transfer" {
					k.SetGuildRankPermissionStoreOnly(ctx, station.Id, guild.Id, types.PermUpdate, 3)
				}
			}
			directID := keeperlib.GetObjectPermissionIDBytes(direct.Id, actor.Id)
			guildID := keeperlib.GetObjectPermissionIDBytes(guild.Id, actor.Id)
			// Preserve the existing kick-specific guild-row cleanup separately
			// from the independent grant on a child object.
			k.SetPermissionsByBytes(ctx, guildID, types.PermPlay)
			vaults, err := k.NewCurrentContext(ctx).EnsureVaults(actor.Id, guild.Id)
			require.NoError(t, err)
			denom := "uguild." + guild.Id
			fundRedelegationAccount(t, bApp, ctx, vaults[0], sdk.NewCoins(sdk.NewInt64Coin(denom, 10)))
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			current, height := ctx, int64(0)
			deliver := func(keys []cryptotypes.PrivKey, messages ...sdk.Msg) *abci.ExecTxResult {
				t.Helper()
				var numbers, sequences []uint64
				for _, key := range keys {
					account := bApp.AccountKeeper.GetAccount(current, sdk.AccAddress(key.PubKey().Address()))
					numbers = append(numbers, account.GetAccountNumber())
					sequences = append(sequences, account.GetSequence())
				}
				if height > 0 {
					_, err := bApp.Commit()
					require.NoError(t, err)
				}
				height++
				tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(height)), config, messages, nil,
					ante.DefaultFreeGasCap, jailGateChainID, numbers, sequences, keys...)
				require.NoError(t, err)
				wire, err := config.TxEncoder()(tx)
				require.NoError(t, err)
				when := ctx.BlockTime().Add(time.Duration(height-1) * time.Second)
				block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: when, Txs: [][]byte{wire}})
				require.NoError(t, err)
				require.Len(t, block.TxResults, 1)
				result := block.TxResults[0]
				require.LessOrEqual(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
				require.NotContains(t, result.Log, "panic")
				current = bApp.NewContext(false).WithBlockTime(when)
				return result
			}
			actorKeys, ownerKeys := []cryptotypes.PrivKey{actorKey}, []cryptotypes.PrivKey{ownerKey}
			grant := &types.MsgPermissionGrantOnObject{Creator: actorAddress.String(), ObjectId: direct.Id, PlayerId: actor.Id, Permissions: uint64(types.PermUpdate)}
			if transition != "ownership_transfer" {
                refused := deliver(actorKeys, grant)
                require.NotZero(t, refused.Code)
                purchase := deliver(actorKeys, &types.MsgGuildRankPurchase{Creator: actorAddress.String(), GuildId: guild.Id, ExpectedRank: 5, TargetRank: 3, MaxPrice: 2})
                require.Zero(t, purchase.Code, purchase.Log)
                refused = deliver(actorKeys, grant)
                require.NotZero(t, refused.Code, "purchased operational rank cannot create permanent grants")
                require.Equal(t, types.Permissionless, k.GetPermissionsByBytes(current, directID))
                grant.Creator = ownerAddress.String()
            }
            grantKeys := ownerKeys
            if transition == "ownership_transfer" { grantKeys = actorKeys }
            result := deliver(grantKeys, grant)
            require.Zero(t, result.Code, result.Log)
			require.Equal(t, types.PermUpdate, k.GetPermissionsByBytes(current, directID))
			price := int64(2)
			if transition == "ownership_transfer" {
				price = 0
			}
			require.Equal(t, int64(10)-price, bApp.BankKeeper.GetBalance(current, vaults[0], denom).Amount.Int64())
			require.Equal(t, price, bApp.BankKeeper.GetBalance(current, vaults[1], denom).Amount.Int64())
			permissionEvent := 0
			for _, event := range result.Events {
				if event.Type == "structs.structs.EventPermission" {
					decoded, err := sdk.ParseTypedEvent(event)
					require.NoError(t, err)
					if decoded.(*types.EventPermission).PermissionRecord.PermissionId == string(directID) {
						require.Equal(t, uint64(types.PermUpdate), decoded.(*types.EventPermission).PermissionRecord.Value)
						permissionEvent++
					}
				}
			}
			require.Equal(t, 1, permissionEvent)
			change := func() []sdk.Msg {
				switch transition {
				case "departure":
					return []sdk.Msg{&types.MsgGuildMembershipJoinByRankPurchase{Creator: actorAddress.String(), GuildId: destination.Id, ExpectedGuildId: guild.Id, TargetRank: 5}}
				case "kick":
					return []sdk.Msg{&types.MsgGuildMembershipKick{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: actor.Id}}
				case "demotion":
					return []sdk.Msg{&types.MsgPlayerUpdateGuildRank{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: actor.Id, GuildRank: 4}}
				case "rank_revoke":
					return []sdk.Msg{
						&types.MsgPermissionGuildRankRevoke{Creator: ownerAddress.String(), GuildId: guild.Id, ObjectId: direct.Id, Permission: uint64(types.PermUpdate)},
						&types.MsgPermissionGuildRankRevoke{Creator: ownerAddress.String(), GuildId: guild.Id, ObjectId: inherited.Id, Permission: uint64(types.PermUpdate)},
					}
				default:
					return []sdk.Msg{&types.MsgGuildUpdateOwnerId{Creator: actorAddress.String(), GuildId: guild.Id, Owner: owner.Id}}
				}
			}
			edit := func(id string) sdk.Msg {
				return &types.MsgSubstationUpdateName{Creator: actorAddress.String(), SubstationId: id, Name: "Changed Station"}
			}
			changeKeys, batchKeys := ownerKeys, []cryptotypes.PrivKey{ownerKey, actorKey}
			if transition == "departure" || transition == "ownership_transfer" {
				changeKeys, batchKeys = actorKeys, actorKeys
			}
			beforePlayer, _ := k.GetPlayer(current, actor.Id)
			beforeGuild, _ := k.GetGuild(current, guild.Id)
			beforePermissions, beforeRanks := k.GetAllPermissionExport(current), k.GetAllGuildRankPermissionExport(current)
			beforeGrid := k.GetAllGridExport(current)
			result = deliver(batchKeys, append(change(), edit(inherited.Id))...)
			require.NotZero(t, result.Code, "authority loss must be visible later in the same batch")
			require.Contains(t, result.Log, fmt.Sprintf("message index: %d", len(change())), "the transition must execute before inherited authority is refused")
			require.Contains(t, result.Log, "permission")
			actualPlayer, _ := k.GetPlayer(current, actor.Id)
			actualGuild, _ := k.GetGuild(current, guild.Id)
			require.Equal(t, beforePlayer, actualPlayer)
			require.Equal(t, beforeGuild, actualGuild)
			require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
			require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
			require.Equal(t, beforeGrid, k.GetAllGridExport(current))
			for _, event := range result.Events {
				require.NotContains(t, event.Type, "structs.structs.Event", "failed messages must not publish rolled-back state")
			}
			result = deliver(changeKeys, change()...)
			require.Zero(t, result.Code, result.Log)
			actualPlayer, _ = k.GetPlayer(current, actor.Id)
			switch transition {
			case "departure":
				require.Equal(t, destination.Id, actualPlayer.GuildId)
				require.Equal(t, destination.EntryRank, actualPlayer.GuildRank)
			case "kick":
				require.Empty(t, actualPlayer.GuildId)
				require.Equal(t, uint64(types.DefaultEntryRank), actualPlayer.GuildRank)
				require.Equal(t, types.PermPlay, k.GetPermissionsByBytes(current, guildID), "kick preserves independent grants")
			case "demotion":
				require.Equal(t, uint64(4), actualPlayer.GuildRank)
			case "ownership_transfer":
				actualGuild, _ = k.GetGuild(current, guild.Id)
				require.Equal(t, owner.Id, actualGuild.Owner)
			}
			require.Equal(t, types.PermUpdate, k.GetPermissionsByBytes(current, directID))
			result = deliver(actorKeys, edit(inherited.Id))
			require.NotZero(t, result.Code)
			result = deliver(actorKeys, edit(direct.Id))
			require.Zero(t, result.Code, result.Log)
			actualStation, found := k.GetSubstation(current, direct.Id)
			require.True(t, found)
			require.Equal(t, guild.Id, actualStation.Owner)
			require.Equal(t, "Changed Station", actualStation.Name)
			result = deliver(ownerKeys, &types.MsgPermissionRevokeOnObject{Creator: ownerAddress.String(), ObjectId: direct.Id, PlayerId: actor.Id, Permissions: uint64(types.PermUpdate)})
			require.Zero(t, result.Code, result.Log)
			require.Equal(t, types.Permissionless, k.GetPermissionsByBytes(current, directID))
			result = deliver(actorKeys, edit(direct.Id))
			require.NotZero(t, result.Code)
			require.Equal(t, int64(10)-price, bApp.BankKeeper.GetBalance(current, vaults[0], denom).Amount.Int64())
			require.Equal(t, price, bApp.BankKeeper.GetBalance(current, vaults[1], denom).Amount.Int64())
			require.Equal(t, int64(10), bApp.BankKeeper.GetSupply(current, denom).Amount.Int64())
			_, err = bApp.Commit()
			require.NoError(t, err)
		})
	}
}
