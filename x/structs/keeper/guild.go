package keeper

import (
	"encoding/binary"

    "context"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

   banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	//"strconv"
	//"strings"
)

// GetGuildCount get the total number of guild
func (k Keeper) GetGuildCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.GuildCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0  {
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

// AppendGuild appends a guild in the store with a new id and update the count
func (k Keeper) AppendGuild(
	ctx context.Context,
	//guild types.Guild,
	endpoint string,
	substationId string,
	reactor types.Reactor,
	player types.Player,
) (guild types.Guild) {
    guild = types.CreateEmptyGuild()

	// Create the guild
	count := k.GetGuildCount(ctx)

	// Set the ID of the appended value
	guild.Id                = GetObjectID(types.ObjectType_guild, count)
	guild.Index             = count
	guild.Endpoint          = endpoint
	guild.Creator           = player.Creator
	guild.Owner             = player.Id
	guild.PrimaryReactorId  = reactor.Id
	guild.EntrySubstationId = substationId

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildKey))
	appendedValue := k.cdc.MustMarshal(&guild)
	store.Set([]byte(guild.Id), appendedValue)

	// Update guild count
	k.SetGuildCount(ctx, count+1)

	permissionId := GetObjectPermissionIDBytes(guild.Id, player.Id)
    k.PermissionAdd(ctx, permissionId, types.PermissionAll)

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

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuild{Guild: &guild})

	return guild
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

