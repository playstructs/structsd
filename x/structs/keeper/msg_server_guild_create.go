package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// GuildCreate is the single paid founding path. The signing player becomes the
// owner; legacy charter proof and third-party consent fields cannot authorize it.
func (k msgServer) GuildCreate(goCtx context.Context, msg *types.MsgGuildCreate) (*types.MsgGuildCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	founder, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return nil, err
	}
	if err := cc.PermissionCheck(founder, founder, types.PermPlay|types.PermTokenTransfer); err != nil {
		return nil, err
	}
	if msg.Proof != "" || msg.Nonce != "" || msg.Address != "" || msg.ProofPubKey != "" || msg.ProofSignature != "" || (msg.FounderPlayerId != "" && msg.FounderPlayerId != founder.ID()) {
		return nil, types.NewParameterValidationError("guild_create", 0, "legacy_charter_fields_not_supported")
	}
	reactor := cc.GetReactor(msg.ReactorId)
	if err := reactor.CheckReactor(); err != nil {
		return nil, types.NewObjectNotFoundError("reactor", msg.ReactorId)
	}
	validatorAddress, err := sdk.ValAddressFromBech32(reactor.GetReactor().Validator)
	if err != nil {
		return nil, types.NewObjectNotFoundError("validator", reactor.GetReactor().Validator)
	}
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	if err != nil {
		return nil, types.NewObjectNotFoundError("validator", validatorAddress.String())
	}
	if validator.IsJailed() {
		return nil, types.NewReactorError("guild_create", "validator_jailed")
	}
	oldGuild, err := guildCharterLeavable(cc, founder)
	if err != nil {
		return nil, err
	}
	if founder.GetSubstationId() != "" {
		if _, err := cc.GetExistingSubstation(founder.GetSubstationId()); err != nil {
			return nil, err
		}
	}
	if msg.EntrySubstationId != "" {
		substation, err := cc.GetExistingSubstation(msg.EntrySubstationId)
		if err != nil {
			return nil, err
		}
		if err := substation.CanManageConnectionsBy(founder); err != nil {
			return nil, err
		}
	}
	// Reject before AppendGuild allocates a guild id: FundCreation repeats
	// these checks, but by then the counter has advanced.
	vault, err := cc.GetVault(founder.ID())
	if err != nil {
		return nil, err
	}
	if k.bankKeeper.SpendableCoin(ctx, vault.Address(), "ualpha").Amount.LT(sdk.NewInt64Coin("ualpha", GuildCreationCostAlpha).Amount) {
		return nil, types.NewPlayerAffordabilityError(founder.ID(), "guild_create", "2000000ualpha")
	}
	if err := k.requireSendableAmount(ctx, sdk.NewCoins(sdk.NewInt64Coin("ualpha", GuildCreationCostAlpha))); err != nil {
		return nil, err
	}
	guild := k.AppendGuild(ctx, msg.Endpoint, msg.EntrySubstationId, reactor.GetReactor(), founder.GetPlayer(), "")
	if err := cc.GetGuild(guild.Id).FundCreation(founder); err != nil {
		return nil, err
	}
	guildCharterLeaveAndJoin(cc, founder, oldGuild, guild.Id, msg.EntrySubstationId)
	// A guild's preference does not claim, control or consume its primary reactor.
	k.AddressEmitActivity(ctx, msg.Creator)
	cc.CommitAll()
	return &types.MsgGuildCreateResponse{GuildId: guild.Id}, nil
}
