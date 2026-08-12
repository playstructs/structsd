package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	"structs/x/structs/types"
)

type ProviderPoolKind byte

const (
	ProviderPoolKindCollateral ProviderPoolKind = 1
	ProviderPoolKindEarnings   ProviderPoolKind = 2
)

// SetProviderPoolAddress records the provider and role behind a derived pool
// address. The address itself is an ADR-028 hash and cannot be reversed, so this
// index is what both the guild-token holder gate and confiscation policy resolve.
func (k Keeper) SetProviderPoolAddress(ctx context.Context, address, providerId string, kind ProviderPoolKind) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderPoolAddressKey))
	value := append([]byte{byte(kind)}, []byte(providerId)...)
	store.Set([]byte(address), value)
}

func (k Keeper) GetProviderPoolAddress(ctx context.Context, address string) (string, ProviderPoolKind, bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderPoolAddressKey))
	value := store.Get([]byte(address))
	if len(value) < 2 {
		return "", 0, false
	}

	kind := ProviderPoolKind(value[0])
	if kind != ProviderPoolKindCollateral && kind != ProviderPoolKindEarnings {
		return "", 0, false
	}

	return string(value[1:]), kind, true
}

func (k Keeper) RemoveProviderPoolAddress(ctx context.Context, address string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderPoolAddressKey))
	store.Delete([]byte(address))
}

func (k Keeper) IndexProviderPoolAddresses(ctx context.Context, providerId string) {
	k.SetProviderPoolAddress(ctx, GetProviderCollateralPoolLocation(providerId).String(), providerId, ProviderPoolKindCollateral)
	k.SetProviderPoolAddress(ctx, GetProviderEarningsPoolLocation(providerId).String(), providerId, ProviderPoolKindEarnings)
}

func (k Keeper) RemoveProviderPoolAddresses(ctx context.Context, providerId string) {
	k.RemoveProviderPoolAddress(ctx, GetProviderCollateralPoolLocation(providerId).String())
	k.RemoveProviderPoolAddress(ctx, GetProviderEarningsPoolLocation(providerId).String())
}

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

// ProtectLegacyGuildEscrowBalances rebuilds the protect-only marker from IBC
// channel state and bank balances. Both are exported by their owning modules, so
// this derived index can be reconstructed during structs genesis import instead
// of adding it to the generated GenesisState.
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
