package types

import (
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
)

const GeneratorPowerReconcileBudget = 128

// ValidateGeneratorPower protects arithmetic before a worker changes accounting.
// A disabled generator retains its ratio and fuel, but contributes no power.
func (a Infusion) ValidateGeneratorPower() error {
	if a.DestinationType != ObjectType_struct {
		return fmt.Errorf("generator power requires a struct infusion")
	}
	if err := a.ValidateAssetOwner(); err != nil {
		return err
	}
	if a.Commission.IsNil() || a.Commission.IsNegative() || a.Commission.GT(math.LegacyOneDec()) {
		return fmt.Errorf("invalid generator commission")
	}
	if a.Ratio != 0 && a.Fuel > ^uint64(0)/a.Ratio {
		return fmt.Errorf("generator power exceeds uint64")
	}
	power := a.Ratio * a.Fuel
	if a.GeneratorDisabled {
		power = 0
	}
	if a.Power != power {
		return fmt.Errorf("generator power does not match fuel and status")
	}
	return nil
}

func (gs GenesisState) ValidateGeneratorPowerQueue() error {
	seen := map[string]bool{}
	for _, id := range gs.GeneratorPowerQueue {
		parts := strings.Split(id, "-")
		if len(parts) != 2 || parts[0] != "5" {
			return fmt.Errorf("invalid generator queue id %q", id)
		}
		index, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil || id != fmt.Sprintf("5-%d", index) {
			return fmt.Errorf("invalid generator queue id %q", id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate generator queue id %q", id)
		}
		seen[id] = true
	}
	return nil
}
