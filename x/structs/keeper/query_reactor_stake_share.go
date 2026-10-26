package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"
)

func (k Keeper) ReactorStakeShare(ctx context.Context, req *types.QueryReactorStakeShareRequest) (*types.QueryReactorStakeShareResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	reactor, found := k.GetReactor(ctx, req.ReactorId)
	if !found {
		return nil, status.Error(codes.NotFound, "reactor not found")
	}
	address, err := sdk.ValAddressFromBech32(reactor.Validator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	validator, err := k.stakingKeeper.GetValidator(ctx, address)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	denominator, err := k.reactorStakeDenominator(ctx, validator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	p := k.GetParams(ctx).WithReactorDefaults()
	if err := p.ValidateReactorParams(); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	revision, err := k.GetReactorCommissionRevision(ctx, reactor.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	result := &types.QueryReactorStakeShareResponse{
		ReactorId: reactor.Id, ValidatorAddress: reactor.Validator,
		ValidatorTokens: validator.Tokens.String(), StakeDenominator: denominator.String(),
		CapBps: p.ReactorStakeCapBps, CapActive: k.ReactorStakeCapActive(ctx),
		SlashRevision: revision.Issued, ReconciledSlashRevision: revision.Certified,
		CommissionRepairPending: revision.Issued != revision.Certified,
	}
	if result.CapActive {
		result.ExternalDepositHeadroom = types.ReactorStakeHeadroom(validator.Tokens.BigInt(), denominator, p.ReactorStakeCapBps, true)
		result.BondedRedelegationHeadroom = types.ReactorStakeHeadroom(validator.Tokens.BigInt(), denominator, p.ReactorStakeCapBps, false)
	}
	return result, nil
}
