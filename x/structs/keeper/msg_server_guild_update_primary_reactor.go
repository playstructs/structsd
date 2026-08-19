package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// GuildUpdatePrimaryReactor reassigns a guild's primary reactor to a different
// reactor. This is the recovery path when the validator backing the current
// primary reactor has been permanently retired (tombstoned, withdrawn, or
// otherwise unrecoverable). The guild's economic surface (token mints, guild
// bank, member infusion routing) is wired through the primary reactor; without
// this op, a guild whose validator is gone has no way to keep operating.
//
// Authorization: PermAdmin on the guild object (see CanUpdatePrimaryReactorBy
// for the rationale on reusing PermAdmin instead of minting a new bit).
//
// Validation:
//
//   - Player exists and is registered.
//   - Guild exists.
//   - Caller has PermAdmin on the guild.
//   - Target reactor exists (resolves to a known reactor object).
//   - Target reactor's validator is registered with x/staking and is not
//     currently jailed. We deliberately omit a Bonded check: a guild may
//     legitimately want to rotate to an unbonded but otherwise healthy
//     validator (e.g. one in the process of bonding for the first time), and
//     we don't want this recovery path to block in that case.
func (k msgServer) GuildUpdatePrimaryReactor(goCtx context.Context, msg *types.MsgGuildUpdatePrimaryReactor) (*types.MsgGuildUpdateResponse, error) {
	emptyResponse := &types.MsgGuildUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Indexer activity record (mirrors the other guild-update handlers).
	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_primary_reactor")
	}

	guild := cc.GetGuild(msg.GuildId)
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

	if permErr := guild.CanUpdatePrimaryReactorBy(player); permErr != nil {
		return emptyResponse, permErr
	}

	reactor := cc.GetReactor(msg.ReactorId)
	if reactor.CheckReactor() != nil {
		return emptyResponse, types.NewObjectNotFoundError("reactor", msg.ReactorId)
	}

	// Validator-side health check: must exist and not be jailed. A jailed
	// validator's reactor would re-create the same outage that motivated the
	// recovery in the first place.
	validatorAddress, err := sdk.ValAddressFromBech32(reactor.GetReactor().Validator)
	if err != nil {
		return emptyResponse, types.NewObjectNotFoundError("validator", reactor.GetReactor().Validator)
	}
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	if err != nil {
		return emptyResponse, types.NewObjectNotFoundError("validator", validatorAddress.String())
	}
	if validator.IsJailed() {
		return emptyResponse, types.NewObjectNotFoundError("validator", validatorAddress.String()+" (jailed)")
	}

	if guild.GetPrimaryReactorId() != msg.ReactorId {
		guild.SetPrimaryReactorId(msg.ReactorId)
	}

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
