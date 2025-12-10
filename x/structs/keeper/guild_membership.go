package keeper

import (
    "context"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//"strconv"
	//"strings"
)


// GetGuildMembershipID returns the string representation of the ID
func GetGuildMembershipApplicationID(guildId string, playerId string) string {
	return playerId + "@" + guildId
}

func (k Keeper) GetGuildMembershipApplication(ctx context.Context, guildId string, playerId string) (val types.GuildMembershipApplication, found bool)  {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildMembershipApplicationKey))
	b := store.Get([]byte(GetGuildMembershipApplicationID(guildId, playerId)))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) SetGuildMembershipApplication(ctx context.Context, guildMembership types.GuildMembershipApplication) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildMembershipApplicationKey))
	b := k.cdc.MustMarshal(&guildMembership)
	store.Set([]byte(GetGuildMembershipApplicationID(guildMembership.GuildId, guildMembership.PlayerId)), b)
}

func (k Keeper) ClearGuildMembershipApplication(ctx context.Context, guildId string, playerId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildMembershipApplicationKey))
	store.Delete([]byte(GetGuildMembershipApplicationID(guildId, playerId)))
}

func (k Keeper) EventGuildMembershipApplication(ctx context.Context, guildMembership types.GuildMembershipApplication) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildMembershipApplication{GuildMembershipApplication: &guildMembership})
}



