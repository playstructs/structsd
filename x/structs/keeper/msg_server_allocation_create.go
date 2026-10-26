package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) AllocationCreate(goCtx context.Context, msg *types.MsgAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
	emptyResponse := &types.MsgAllocationCreateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return emptyResponse, err
	}

	sourceObject := cc.GetPermissionedObject(msg.SourceObjectId)
	if sourceObject == nil {
		return emptyResponse, types.NewAllocationError(msg.SourceObjectId, "unacceptable_source")
	}
	permissionErr := sourceObject.CanAllocateAsSourceBy(callingPlayer)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}
	sourceGuild := cc.allocationSourceGuild(sourceObject)
	if msg.Controller == "" {
		msg.Controller = callingPlayer.ID()
		if sourceGuild != nil {
			msg.Controller = sourceGuild.ID()
		}
	}
	controller, err := cc.GetExistingAssetOwner(msg.Controller)
	if err != nil {
		return emptyResponse, err
	}
	// Operational access to guild power must not mint personal administrative
	// authority that outlives the member's rank. Exporting control is an admin act.
	if sourceGuild != nil && msg.Controller != sourceGuild.ID() {
		if err := cc.PermissionAdministrationCheck(sourceObject, callingPlayer); err != nil {
			return emptyResponse, err
		}
	}

	allocation, err := cc.NewAllocation(
		msg.AllocationType,
		msg.SourceObjectId,
		"",
		msg.Creator,
		msg.Controller,
		msg.Power,
	)
	if err != nil {
		return emptyResponse, err
	}

	var creatorPermissions types.Permission
	if allocation.IsAutomated() {
		creatorPermissions = types.PermDelete
	}

	if allocation.IsDynamic() {
		creatorPermissions = types.PermUpdate | types.PermDelete
	}

	if callingPlayer.ID() == msg.Controller {
		creatorPermissions = creatorPermissions | types.PermAllocationConnection | types.PermAdmin
	} else if _, playerController := controller.(*PlayerCache); playerController {
		allocationPermissionId := GetObjectPermissionIDBytes(allocation.ID(), msg.Controller)
		cc.SetPermissions(allocationPermissionId, types.PermAllocationConnection|types.PermAdmin)
	}

	// Guild ownership follows the guild's live owner and explicit rank grants.
	// Do not leave an automatic personal grant with the creating owner.
	_, guildController := controller.(*GuildCache)
	if creatorPermissions != types.Permissionless && !guildController {
		allocationPermissionId := GetObjectPermissionIDBytes(allocation.ID(), callingPlayer.ID())
		cc.SetPermissions(allocationPermissionId, creatorPermissions)
	}

	cc.CommitAll()
	return &types.MsgAllocationCreateResponse{
		AllocationId: allocation.GetAllocationId(),
	}, nil

}
