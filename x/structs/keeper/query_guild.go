package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"

    //"encoding/binary"
    //"strings"
    //"strconv"
    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (k Keeper) GuildAll(goCtx context.Context, req *types.QueryAllGuildRequest) (*types.QueryAllGuildResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var guilds []types.Guild
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	guildStore := prefix.NewStore(store, types.KeyPrefix(types.GuildKey))

	pageRes, err := query.Paginate(guildStore, req.Pagination, func(key []byte, value []byte) error {
		var guild types.Guild
		if err := k.cdc.Unmarshal(value, &guild); err != nil {
			return err
		}

		guilds = append(guilds, guild)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGuildResponse{Guild: guilds, Pagination: pageRes}, nil
}

func (k Keeper) Guild(goCtx context.Context, req *types.QueryGetGuildRequest) (*types.QueryGetGuildResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	guild, found := k.GetGuild(ctx, req.Id)
	if !found {
		return nil, types.ErrObjectNotFound
	}

	return &types.QueryGetGuildResponse{Guild: guild}, nil
}


func (k Keeper) GuildMembershipApplication(goCtx context.Context, req *types.QueryGetGuildMembershipApplicationRequest) (*types.QueryGetGuildMembershipApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	guildMembershipApplication, found := k.GetGuildMembershipApplication(ctx, req.GuildId, req.PlayerId)
	if !found {
		return nil, types.ErrObjectNotFound
	}

	return &types.QueryGetGuildMembershipApplicationResponse{GuildMembershipApplication: guildMembershipApplication}, nil
}



func (k Keeper) GuildMembershipApplicationAll(goCtx context.Context, req *types.QueryAllGuildMembershipApplicationRequest) (*types.QueryAllGuildMembershipApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var guildMembershipApplications []types.GuildMembershipApplication

    ctx := sdk.UnwrapSDKContext(goCtx)
    store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	guildMembershipApplicationStore := prefix.NewStore(store, types.KeyPrefix(types.GuildMembershipApplicationKey))

	pageRes, err := query.Paginate(guildMembershipApplicationStore, req.Pagination, func(key []byte, value []byte) error {
		var guildMembershipApplication types.GuildMembershipApplication

       	if err := k.cdc.Unmarshal(value, &guildMembershipApplication); err != nil {
            return err
        }
        guildMembershipApplications = append(guildMembershipApplications, guildMembershipApplication)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGuildMembershipApplicationResponse{GuildMembershipApplication: guildMembershipApplications, Pagination: pageRes}, nil
}

func (k Keeper) GuildBankCollateralAddress(goCtx context.Context, req *types.QueryGetGuildBankCollateralAddressRequest) (*types.QueryAllGuildBankCollateralAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var addresses []*types.InternalAddressAssociation
    address := authtypes.NewModuleAddress(types.GuildBankCollateralPool + req.GuildId).String()
    addressAssociation := types.InternalAddressAssociation{Address: address, ObjectId: req.GuildId}
    addresses = append(addresses, &addressAssociation)

    return &types.QueryAllGuildBankCollateralAddressResponse{InternalAddressAssociation: addresses}, nil
}


func (k Keeper) GuildBankCollateralAddressAll(goCtx context.Context, req *types.QueryAllGuildBankCollateralAddressRequest) (*types.QueryAllGuildBankCollateralAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var addresses []*types.InternalAddressAssociation
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	guildStore := prefix.NewStore(store, types.KeyPrefix(types.GuildKey))

	pageRes, err := query.Paginate(guildStore, req.Pagination, func(key []byte, value []byte) error {
		var guild types.Guild
		if err := k.cdc.Unmarshal(value, &guild); err != nil {
			return err
		}

        address := authtypes.NewModuleAddress(types.GuildBankCollateralPool + guild.Id).String()
        addressAssociation := types.InternalAddressAssociation{Address: address, ObjectId: guild.Id}
        addresses = append(addresses, &addressAssociation)

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGuildBankCollateralAddressResponse{InternalAddressAssociation: addresses, Pagination: pageRes}, nil
}


