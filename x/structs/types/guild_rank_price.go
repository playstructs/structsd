package types

import "math/big"

// A zero minimum means sales are disabled. Admission cannot be enabled without
// a valid purchasable range. Legacy zero settings remain a valid disabled state.
func ValidateRankPurchaseSettings(entryRank, minimumRank uint64, joinEnabled bool) error {
	if minimumRank == 0 {
		if joinEnabled {
			return NewParameterValidationError("minimum_rank", 0, "required_for_purchase_admission")
		}
		return nil
	}
	if entryRank == 0 || minimumRank > entryRank {
		return NewParameterValidationError("minimum_rank", minimumRank, "exceeds_entry_rank")
	}
	return nil
}

// CalculateGuildRankPurchasePrice prices a move to a strictly better numeric
// rank in base units of the guild token. Progress is measured above entryRank;
// ranks worse than entryRank have no paid progress. Admission can quote from
// entryRank itself. Authorization and the purchase ceiling belong to the caller.
//
// The nth step costs basePrice + (n-1)*stepIncrease. Calculate the difference
// using wide integers: either cumulative price can exceed uint64 even when the
// individual purchase fits. Only the final payable amount must fit uint64.
func CalculateGuildRankPurchasePrice(entryRank, currentRank, targetRank, basePrice, stepIncrease uint64) (uint64, error) {
	if entryRank == 0 || currentRank == 0 || targetRank == 0 {
		return 0, NewParameterValidationError("rank", 0, "must_be_positive")
	}
	if targetRank >= currentRank || targetRank > entryRank {
		return 0, NewParameterValidationError("target_rank", targetRank, "must_improve_rank_within_entry_range")
	}
	progress := func(rank uint64) uint64 {
		if rank >= entryRank {
			return 0
		}
		return entryRank - rank
	}
	total := func(n uint64) *big.Int {
		steps := new(big.Int).SetUint64(n)
		result := new(big.Int).Mul(steps, new(big.Int).SetUint64(basePrice))
		if n > 1 && stepIncrease != 0 {
			increases := new(big.Int).Mul(steps, new(big.Int).SetUint64(n-1))
			increases.Mul(increases, new(big.Int).SetUint64(stepIncrease))
			increases.Quo(increases, big.NewInt(2))
			result.Add(result, increases)
		}
		return result
	}
	price := new(big.Int).Sub(total(progress(targetRank)), total(progress(currentRank)))
	if !price.IsUint64() {
		return 0, NewParameterValidationError("rank_price", 0, "out_of_range")
	}
	return price.Uint64(), nil
}
