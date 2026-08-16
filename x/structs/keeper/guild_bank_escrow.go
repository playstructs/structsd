package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	"structs/x/structs/types"
)

func (k Keeper) SetLegacyGuildBankEscrow(ctx context.Context, address string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildBankLegacyEscrowKey))
	store.Set([]byte(address), []byte{1})
}

func (k Keeper) IsLegacyGuildBankEscrow(ctx context.Context, address string) bool {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildBankLegacyEscrowKey))
	return store.Has([]byte(address))
}

func (k Keeper) RemoveLegacyGuildBankEscrow(ctx context.Context, address string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GuildBankLegacyEscrowKey))
	store.Delete([]byte(address))
}

// ProtectLegacyGuildEscrowBalances rebuilds the protect-only marker from
// exported IBC channel and bank state.
func (k Keeper) ProtectLegacyGuildEscrowBalances(ctx context.Context) {
	if k.ibcKeeperFn == nil {
		return
	}
	ibcKeeper := k.ibcKeeperFn()
	if ibcKeeper == nil {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	channels := ibcKeeper.ChannelKeeper.GetAllChannelsWithPortPrefix(sdkCtx, ibctransfertypes.PortID)
	for _, channel := range channels {
		if channel.PortId != ibctransfertypes.PortID {
			continue
		}

		escrow := ibctransfertypes.GetEscrowAddress(channel.PortId, channel.ChannelId)
		for _, coin := range k.bankKeeper.GetAllBalances(ctx, escrow) {
			if !types.IsGuildBankDenom(coin.Denom) {
				continue
			}

			k.SetLegacyGuildBankEscrow(ctx, escrow.String())
			k.Logger().Error("Legacy IBC escrow holds guild tokens",
				"channel", channel.ChannelId,
				"address", escrow.String(),
				"denom", coin.Denom,
				"amount", coin.Amount.String(),
			)
		}
	}
}
