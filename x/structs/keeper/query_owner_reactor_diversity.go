package keeper

import (
	"context"
	"math/big"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"
)

func (k Keeper) OwnerReactorDiversity(ctx context.Context, req *types.QueryOwnerReactorDiversityRequest) (*types.QueryOwnerReactorDiversityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if _, _, err := types.ParseAssetOwnerID(req.OwnerId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	cc := k.NewCurrentContext(ctx)
	if _, err := cc.GetExistingAssetOwner(req.OwnerId); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	stored, found := k.GetOwnerReactorDiversity(ctx, req.OwnerId)
	if !found {
		stored = types.OwnerReactorDiversity{OwnerId: req.OwnerId, Power: "0", SquaredPower: "0"}
	}
	for _, value := range []string{stored.Power, stored.SquaredPower} {
		n, ok := new(big.Int).SetString(value, 10)
		if !ok || n.Sign() < 0 || n.String() != value {
			return nil, status.Error(codes.Internal, "invalid diversity totals")
		}
	}
	if stored.OwnerId != req.OwnerId {
		return nil, status.Error(codes.Internal, "diversity owner key mismatch")
	}
	p := k.GetParams(ctx).WithReactorDefaults()
	if err := p.ValidateReactorParams(); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	result := &types.QueryOwnerReactorDiversityResponse{
		Stored: &stored, MaxBonusBps: p.DiversityMaxBonusBps, TargetExtraReactors: p.DiversityTargetExtraReactors,
		Initialized: k.ReactorDiversityReady(ctx), StoredCapacity: cc.GetGridAttribute(GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, req.OwnerId)),
	}
	portfolio := liveOwnerReactorPortfolio{}
	if result.Initialized {
		portfolio = cc.liveOwnerReactorPortfolio(req.OwnerId)
	}
	result.UsablePrimaryCapacity, _, result.InfusionInventoryComplete = cc.getUsableOwnerCapacityProofWithPortfolio(req.OwnerId, OwnerSupplyReadBudget, &portfolio)
	result.LivePortfolioComplete = portfolio.complete
	if portfolio.complete {
		result.LiveBonus = portfolio.bonus.String()
		if result.InfusionInventoryComplete {
			result.UsableBonus = stored.AppliedBonus
			if portfolio.bonus.IsUint64() {
				result.UsableBonus = min(result.UsableBonus, portfolio.bonus.Uint64())
			}
		}
	}
	return result, nil
}
