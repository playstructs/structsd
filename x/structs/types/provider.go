package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
)



func CreateBaseProvider(creator string, owner string) (Provider) {
    return Provider{
        Creator: creator,
        Owner: owner,

    }
}

func (provider *Provider) SetCapacityRange(minimum uint64, maximum uint64) error {
    if minimum > maximum {
        return sdkerrors.Wrapf(ErrInvalidParameters, "Minimum Capacity (%d) cannot be larger than Maximum Capacity (%d)", minimum, maximum)
    }

    provider.CapacityMinimum = minimum
    provider.CapacityMaximum = maximum
	return nil
}

func (provider *Provider) SetCapacityMaximum(maximum uint64) error {
    if provider.CapacityMinimum > maximum {
        return sdkerrors.Wrapf(ErrInvalidParameters, "Minimum Capacity (%d) cannot be larger than Maximum Capacity (%d)", provider.CapacityMinimum, maximum)
    }
    provider.CapacityMaximum = maximum
	return nil
}

func (provider *Provider) SetCapacityMinimum(minimum uint64) error {
    if minimum > provider.CapacityMaximum {
        return sdkerrors.Wrapf(ErrInvalidParameters, "Minimum Capacity (%d) cannot be larger than Maximum Capacity (%d)", minimum, provider.CapacityMaximum)
    }
    provider.CapacityMinimum = minimum
	return nil
}


func (provider *Provider) SetDurationRange(minimum uint64, maximum uint64) error {
    if minimum > maximum {
        return sdkerrors.Wrapf(ErrInvalidParameters, "Minimum Duration (%d) cannot be larger than Maximum Duration (%d)", minimum, maximum)
    }

    provider.DurationMinimum = minimum
    provider.DurationMaximum = maximum
	return nil
}

func (provider *Provider) SetDurationMaximum(maximum uint64) error {
    if provider.DurationMinimum > maximum {
        return sdkerrors.Wrapf(ErrInvalidParameters, "Minimum Duration (%d) cannot be larger than Maximum Duration (%d)", provider.DurationMinimum, maximum)
    }
    provider.DurationMaximum = maximum
	return nil
}

func (provider *Provider) SetDurationMinimum(minimum uint64) error {
    if minimum > provider.DurationMaximum {
        return sdkerrors.Wrapf(ErrInvalidParameters, "Minimum Duration (%d) cannot be larger than Maximum Duration (%d)", minimum, provider.DurationMaximum)
    }
    provider.DurationMinimum = minimum
	return nil
}


func (provider *Provider) SetProviderCancellationPenalty(penalty math.LegacyDec) error {
    one, _ := math.LegacyNewDecFromStr("1")

    // 1 <= Provider Cancellation Policy => 0
    if penalty.GTE(math.LegacyZeroDec()) && penalty.LTE(one) {
        return sdkerrors.Wrapf(ErrInvalidParameters, "Provider Cancellation Penalty (%f) must be between 1 and 0", penalty)
    }

    provider.ProviderCancellationPenalty = penalty
	return nil
}


func (provider *Provider) SetConsumerCancellationPenalty(penalty math.LegacyDec) error {
    one, _ := math.LegacyNewDecFromStr("1")

    // 1 <= Provider Cancellation Policy => 0
    if penalty.GTE(math.LegacyZeroDec()) && penalty.LTE(one) {
        return sdkerrors.Wrapf(ErrInvalidParameters, "Consumer Cancellation Penalty (%f) must be between 1 and 0", penalty)
    }

    provider.ConsumerCancellationPenalty = penalty
	return nil
}

func (provider *Provider) SetAccessPolicy(accessPolicy ProviderAccessPolicy) {
    provider.AccessPolicy = accessPolicy
}
