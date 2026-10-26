package keeper

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// These full-store readers belong only to explicit reconstruction/invariant
// paths. Runtime changes use the two directly keyed CurrentContext caches.
func (k Keeper) allReactorDiversityPositions(ctx context.Context) ([]types.ReactorDiversityPosition, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), reactorDiversityPositionPrefix)
	it := store.Iterator(nil, nil)
	defer it.Close()
	var result []types.ReactorDiversityPosition
	for ; it.Valid(); it.Next() {
		var row types.ReactorDiversityPosition
		if err := k.cdc.Unmarshal(it.Value(), &row); err != nil {
			return nil, err
		}
		if string(it.Key()) != row.OwnerId+"/"+row.ReactorId {
			return nil, fmt.Errorf("diversity position key mismatch")
		}
		power, ok := new(big.Int).SetString(row.Power, 10)
		if !ok || power.Sign() <= 0 || power.String() != row.Power {
			return nil, fmt.Errorf("invalid diversity position power")
		}
		result = append(result, row)
	}
	return result, nil
}

func (k Keeper) allOwnerReactorDiversity(ctx context.Context) ([]types.OwnerReactorDiversity, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), ownerReactorDiversityPrefix)
	it := store.Iterator(nil, nil)
	defer it.Close()
	var result []types.OwnerReactorDiversity
	for ; it.Valid(); it.Next() {
		var row types.OwnerReactorDiversity
		if err := k.cdc.Unmarshal(it.Value(), &row); err != nil {
			return nil, err
		}
		if string(it.Key()) != row.OwnerId {
			return nil, fmt.Errorf("diversity owner key mismatch")
		}
		for _, value := range []string{row.Power, row.SquaredPower} {
			n, ok := new(big.Int).SetString(value, 10)
			if !ok || n.Sign() < 0 || n.String() != value {
				return nil, fmt.Errorf("invalid diversity owner totals")
			}
		}
		result = append(result, row)
	}
	return result, nil
}

func validateDiversityInfusion(row types.Infusion) error {
	if err := row.ValidateAssetOwner(); err != nil {
		return err
	}
	if _, _, err := types.ParseAssetOwnerID(row.AssetOwnerID()); err != nil {
		return err
	}
	if row.Commission.IsNil() || row.Commission.IsNegative() || row.Commission.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("invalid infusion commission for diversity")
	}
	if row.DestinationType != types.ObjectType_reactor && row.DestinationType != types.ObjectType_struct {
		return fmt.Errorf("invalid infusion source for diversity")
	}
	if !ObjectIdHasType(row.DestinationId, row.DestinationType) {
		return fmt.Errorf("invalid infusion destination for diversity")
	}
	return nil
}

// RebuildReactorDiversity runs after final infusion ownership/base-power repair.
// It withdraws any previously applied bonus, replaces all current derived rows,
// then applies the recomputed bonus once. A private SDK cache makes even direct
// invocations atomic; the upgrade's outer cache still owns publication.
func (k Keeper) RebuildReactorDiversity(ctx context.Context) error {
	work, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	positions, err := k.allReactorDiversityPositions(work)
	if err != nil {
		return err
	}
	owners, err := k.allOwnerReactorDiversity(work)
	if err != nil {
		return err
	}
	p := k.GetParams(work).WithReactorDefaults()
	if err := p.ValidateReactorParams(); err != nil {
		return err
	}
	cc := k.NewCurrentContext(work)
	before := make(map[string]uint64)
	for _, row := range positions {
		position := cc.GetReactorDiversityPosition(row.OwnerId, row.ReactorId)
		position.Power.SetUint64(0)
		position.Changed = true
	}
	for _, row := range owners {
		key := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, row.OwnerId)
		total := k.GetGridAttribute(work, key)
		if total < row.AppliedBonus {
			return fmt.Errorf("diversity exceeds capacity for %s", row.OwnerId)
		}
		before[row.OwnerId] = total
		cc.SetGridAttribute(key, total-row.AppliedBonus)
		summary := cc.GetOwnerReactorDiversity(row.OwnerId)
		summary.Power.SetUint64(0)
		summary.SquaredPower.SetUint64(0)
		summary.AppliedBonus = 0
		summary.Changed, summary.Dirty = true, true
	}
	for _, row := range k.GetAllInfusion(work) {
		if err := validateDiversityInfusion(row); err != nil {
			return err
		}
		if row.DestinationType != types.ObjectType_reactor {
			continue
		}
		owner := row.AssetOwnerID()
		if _, err := cc.GetExistingAssetOwner(owner); err != nil {
			return err
		}
		_, _, power := row.GetPowerDistribution()
		if _, found := before[owner]; !found {
			before[owner] = k.GetGridAttribute(work, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner))
		}
		cc.ChangeReactorDiversityPosition(owner, row.DestinationId, 0, power)
	}
	keys := make([]string, 0, len(before))
	for owner := range before {
		keys = append(keys, owner)
	}
	sort.Strings(keys)
	for _, owner := range keys {
		cc.refreshOwnerReactorDiversity(owner, false)
		cc.propagateOwnerCapacityChange(owner, before[owner], cc.getGridAttribute(GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner)))
	}
	cc.CommitAll()
	runtime.KVStoreAdapter(k.storeService.OpenKVStore(work)).Set(reactorDiversityReadyKey, []byte{1})
	write()
	return nil
}

// CheckReactorDiversityAccounting independently reconstructs reactor aggregates
// and all infusion-backed owner base power. It checks stored accounting only;
// live validator eligibility remains the responsibility of readiness checks.
func (k Keeper) CheckReactorDiversityAccounting(ctx context.Context) error {
	positions, err := k.allReactorDiversityPositions(ctx)
	if err != nil {
		return err
	}
	owners, err := k.allOwnerReactorDiversity(ctx)
	if err != nil {
		return err
	}
	byOwner := make(map[string]map[string]*big.Int)
	base := make(map[string]*big.Int)
	for _, row := range k.GetAllInfusion(ctx) {
		if err := validateDiversityInfusion(row); err != nil {
			return err
		}
		_, _, power := row.GetPowerDistribution()
		owner := row.AssetOwnerID()
		if base[owner] == nil {
			base[owner] = new(big.Int)
		}
		base[owner].Add(base[owner], new(big.Int).SetUint64(power))
		if power == 0 || row.DestinationType != types.ObjectType_reactor {
			continue
		}
		if byOwner[owner] == nil {
			byOwner[owner] = make(map[string]*big.Int)
		}
		if byOwner[owner][row.DestinationId] == nil {
			byOwner[owner][row.DestinationId] = new(big.Int)
		}
		byOwner[owner][row.DestinationId].Add(byOwner[owner][row.DestinationId], new(big.Int).SetUint64(power))
	}
	expectedPositions := make(map[string]string)
	expectedOwners := make(map[string]types.OwnerReactorDiversity)
	p := k.GetParams(ctx).WithReactorDefaults()
	if err := p.ValidateReactorParams(); err != nil {
		return err
	}
	for owner, reactors := range byOwner {
		sum, squares := new(big.Int), new(big.Int)
		for reactor, power := range reactors {
			expectedPositions[owner+"/"+reactor] = power.String()
			sum.Add(sum, power)
			squares.Add(squares, new(big.Int).Mul(power, power))
		}
		bonus, err := types.CalculateReactorDiversityBonus(sum, squares, p.DiversityMaxBonusBps, p.DiversityTargetExtraReactors)
		if err != nil {
			return err
		}
		room := new(big.Int).Sub(new(big.Int).SetUint64(math.MaxUint64), base[owner])
		if room.Sign() < 0 {
			room.SetUint64(0)
		}
		if bonus.Cmp(room) > 0 {
			bonus.Set(room)
		}
		expectedOwners[owner] = types.OwnerReactorDiversity{OwnerId: owner, Power: sum.String(), SquaredPower: squares.String(), AppliedBonus: bonus.Uint64()}
	}
	if len(positions) != len(expectedPositions) || len(owners) != len(expectedOwners) {
		return fmt.Errorf("reactor diversity inventory mismatch")
	}
	for _, row := range positions {
		if expectedPositions[row.OwnerId+"/"+row.ReactorId] != row.Power {
			return fmt.Errorf("reactor diversity position mismatch for %s/%s", row.OwnerId, row.ReactorId)
		}
	}
	for _, row := range owners {
		expected, exists := expectedOwners[row.OwnerId]
		if !exists || row.Power != expected.Power || row.SquaredPower != expected.SquaredPower || row.AppliedBonus != expected.AppliedBonus {
			return fmt.Errorf("reactor diversity summary mismatch for %s", row.OwnerId)
		}
		total := new(big.Int).Add(base[row.OwnerId], new(big.Int).SetUint64(row.AppliedBonus))
		if total.Cmp(new(big.Int).SetUint64(math.MaxUint64)) > 0 {
			total.SetUint64(math.MaxUint64)
		}
		if k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, row.OwnerId)) != total.Uint64() {
			return fmt.Errorf("reactor diversity capacity mismatch for %s", row.OwnerId)
		}
	}
	return nil
}
