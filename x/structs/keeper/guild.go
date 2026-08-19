package keeper

import (
	"encoding/binary"

	"context"

	"structs/x/structs/types"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	//"strconv"
	//"strings"
	//"fmt"
)

// GetGuildCount get the total number of guild
func (k Keeper) GetGuildCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.GuildCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetGuildCount set the total number of guild
func (k Keeper) SetGuildCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.GuildCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

/* AppendGuild appends a guild in the store with a new id and update the count.
 *
 * player is the founder, who becomes Owner and whose address becomes Creator
 * even when somebody else signed the transaction. charterSolverId credits
 * whoever solved the proof-of-work and is empty for a guild founded through a
 * reactor's entitlement. It is taken here rather than through a cache setter
 * because it is written once and never mutated, so credit cannot drift after a
 * transfer.
 */
func (k Keeper) AppendGuild(
	ctx context.Context,
	//guild types.Guild,
	endpoint string,
	substationId string,
	reactor types.Reactor,
	player types.Player,
	charterSolverId string,
) (guild types.Guild) {
	guild = types.CreateEmptyGuild()

	// Create the guild
	count := k.GetGuildCount(ctx)

	// Set the ID of the appended value
	guild.Id = GetObjectID(types.ObjectType_guild, count)
	guild.Index = count
	guild.Endpoint = endpoint
	guild.Creator = player.Creator
	guild.Owner = player.Id
	guild.PrimaryReactorId = reactor.Id
	guild.EntrySubstationId = substationId
	guild.EntryRank = types.DefaultEntryRank
	guild.CharterSolverId = charterSolverId

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildKey))
	appendedValue := k.cdc.MustMarshal(&guild)
	store.Set([]byte(guild.Id), appendedValue)

	// Update guild count
	k.SetGuildCount(ctx, count+1)

	permissionId := GetObjectPermissionIDBytes(guild.Id, player.Id)
	k.SetPermissionsByBytes(ctx, permissionId, types.PermGuildAll)

	// Setup the Guild Token
	guildDenomMetadata := banktypes.Metadata{
		Name:        "guild." + guild.Id,
		Symbol:      "guild." + guild.Id,
		Description: "The currency of Guild " + guild.Id,
		DenomUnits: []*banktypes.DenomUnit{
			{"uguild." + guild.Id, uint32(0), nil},
			{"guild." + guild.Id, uint32(6), nil},
		},
		Base:    "uguild." + guild.Id,
		Display: "uguild." + guild.Id,
	}

	k.bankKeeper.SetDenomMetaData(ctx, guildDenomMetadata)

	guildCollateralAddressRaw := types.GuildBankCollateralPool+guild.Id
	guildCollateralAddress := authtypes.NewModuleAddress(guildCollateralAddressRaw)
	guildCollateralAccount := k.accountKeeper.NewAccountWithAddress(ctx, guildCollateralAddress)
	guildCollateralAddressStr := guildCollateralAddress.String()

	k.logger.Info("Guild Collateral Pool Address", "raw", guildCollateralAddressRaw, "address", guildCollateralAddressStr)
    	// types.ModuleName Guild Bank Mint

	k.accountKeeper.SetAccount(ctx, guildCollateralAccount)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuild{Guild: &guild})
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankAddress{&types.EventGuildBankAddressDetail{GuildId: guild.Id, BankCollateralPool: guildCollateralAddressStr, BankTokenPool: types.ModuleName}})

	return guild
}

/* GetGuildCharterAnchor returns the height the charter difficulty is measured
 * from: the height of the last proof-founded guild.
 *
 * A missing row returns false rather than zero, because the two mean opposite
 * things. Zero age is the hardest point on the curve, but a caller that reads a
 * missing anchor as height zero gets an age of the entire chain, which is the
 * easiest. Callers must decide explicitly; see CharterAnchor on the context.
 */
func (k Keeper) GetGuildCharterAnchor(ctx context.Context) (uint64, bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	bz := store.Get(types.KeyPrefix(types.GuildCharterAnchorKey))
	if bz == nil {
		return 0, false
	}

	return binary.BigEndian.Uint64(bz), true
}

/* SetGuildCharterAnchor stamps the charter anchor.
 *
 * Called from the proof branch of GuildCreate and nowhere else in the live
 * path. Deliberately not from AppendGuild: the reactor entitlement founds a
 * guild too, and resetting there would wipe every pool's in-flight mining
 * because a validator collected a perk. Genesis import and the upgrade handler
 * also call it, which is why it is a plain setter rather than "reset to now".
 */
func (k Keeper) SetGuildCharterAnchor(ctx context.Context, height uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height)
	store.Set(types.KeyPrefix(types.GuildCharterAnchorKey), bz)
}

// SetGuild set a specific guild in the store
func (k Keeper) SetGuild(ctx context.Context, guild types.Guild) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildKey))
	b := k.cdc.MustMarshal(&guild)
	store.Set([]byte(guild.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuild{Guild: &guild})
}

// GetGuild returns a guild from its id
func (k Keeper) GetGuild(ctx context.Context, guildId string) (val types.Guild, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildKey))
	b := store.Get([]byte(guildId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveGuild removes a guild from the store
func (k Keeper) RemoveGuild(ctx context.Context, guildId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildKey))
	store.Delete([]byte(guildId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: guildId})
}

func (k Keeper) SetGuildNameIndex(ctx context.Context, name string, guildId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildNameKey))
	store.Set([]byte(types.NormalizeName(name)), []byte(guildId))
}

func (k Keeper) RemoveGuildNameIndex(ctx context.Context, name string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildNameKey))
	store.Delete([]byte(types.NormalizeName(name)))
}

func (k Keeper) GetGuildIdByName(ctx context.Context, name string) (string, bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildNameKey))
	bz := store.Get([]byte(types.NormalizeName(name)))
	if bz == nil {
		return "", false
	}
	return string(bz), true
}

// GetAllGuildNameIndex returns the guild name index as normalized key to guild
// id.
//
// Every other accessor here reaches a row by re-normalizing a name, which only
// finds rows whose key the current normalization still produces. This walks the
// prefix instead, so it also sees rows that no name maps to any more. Only the
// v0.21.0 index rebuild needs that view.
func (k Keeper) GetAllGuildNameIndex(ctx context.Context) map[string]string {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildNameKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	index := make(map[string]string)
	for ; iterator.Valid(); iterator.Next() {
		index[string(iterator.Key())] = string(iterator.Value())
	}

	return index
}

// ClearGuildNameIndex removes every row in the guild name index and returns how
// many it removed.
//
// Reserved for the v0.21.0 rebuild, which has to drop rows it cannot address by
// name. Ordinary maintenance goes through RemoveGuildNameIndex so that the
// index stays in step with the guild being changed.
func (k Keeper) ClearGuildNameIndex(ctx context.Context) int {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildNameKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	// Collect before deleting: mutating the store under a live iterator is
	// undefined.
	keys := make([][]byte, 0)
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	iterator.Close()

	for _, key := range keys {
		store.Delete(key)
	}

	return len(keys)
}

// GetAllGuild returns all guild
func (k Keeper) GetAllGuild(ctx context.Context) (list []types.Guild) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Guild
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
