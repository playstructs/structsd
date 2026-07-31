package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

/* ReactorRestart reconciles a reactor's infusions against live Cosmos staking
 * state, recomputing every delegator's fuel and deriving the energy ratio from
 * validator health.
 *
 * This exists because reactor energy is gated to zero while a validator sits in
 * jail. A routine unjail recovers on its own, since staking rebonds the
 * validator at the end of that block and AfterValidatorBonded restores the
 * reactor. An operator who unjails while below the active-set cutoff is never
 * rebonded, that hook never fires, and without this message their reactor would
 * stay dark with no way to revive it.
 *
 * Deliberately carries no ownership or permission check. Reconciliation only
 * ever writes state derived from the staking module, so the strongest thing a
 * caller can do is spend their own gas making the grid agree with Cosmos. The
 * same property makes it useful in the opposite direction: anyone can use it to
 * force-gate a jailed reactor whose automatic gate was somehow missed.
 *
 * Note that the ante handler still requires the signer to be a registered
 * player, so this is open to any player rather than to any address.
 */
func (k msgServer) ReactorRestart(goCtx context.Context, msg *types.MsgReactorRestart) (*types.MsgReactorRestartResponse, error) {
	emptyResponse := &types.MsgReactorRestartResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	valAddr, valErr := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if valErr != nil {
		return emptyResponse, types.NewReactorError("restart", "invalid_address").WithAddress(msg.ValidatorAddress, "validator")
	}

	if _, reactorFound := k.GetReactorBytesFromValidator(ctx, valAddr.Bytes()); !reactorFound {
		return emptyResponse, types.NewObjectNotFoundError("reactor", msg.ValidatorAddress)
	}

	k.ReactorRestoreEnergy(ctx, valAddr)

	return &types.MsgReactorRestartResponse{}, nil
}
