package keeper_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Compare incremental cache changes with a full independent portfolio rebuild.
// Multiple address slots on the same reactor must be summed before squaring;
// generator contributions affect base capacity but never the bonus formula.
func FuzzDiversityIncrementalPortfolio(f *testing.F) {
	f.Add([]byte{8, 0, 1, 100, 8, 1, 1, 100, 8, 0, 1, 0, 8, 1, 1, 0})
	seed := make([]byte, 512)
	for i := range seed {
		seed[i] = byte(i*73 + i/7)
	}
	f.Add(seed)
	f.Add([]byte("0Y00")) // generator-only portfolio
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 2048 {
			input = input[:2048]
		}
		k, _, goCtx := setupMsgServer(t)
		ctx := sdk.UnwrapSDKContext(goCtx)
		owners := []string{"1-1", "0-1"}
		var powers [2][9][3]uint64 // slot 8 represents generator output
		cc := k.NewCurrentContext(ctx)
		check := func() {
			for owner, id := range owners {
				sum, squares := new(big.Int), new(big.Int)
				base := uint64(0)
				for reactor, addresses := range powers[owner] {
					power := addresses[0] + addresses[1] + addresses[2]
					base += power
					if reactor == 8 {
						continue
					}
					value := new(big.Int).SetUint64(power)
					sum.Add(sum, value)
					squares.Add(squares, new(big.Int).Mul(value, value))
					position := cc.GetReactorDiversityPosition(id, fmt.Sprintf("3-%d", reactor+1))
					require.Equal(t, fmt.Sprint(power), position.Power.String())
				}
				bonus := new(big.Int)
				if sum.Sign() != 0 {
					effective := new(big.Rat).SetFrac(new(big.Int).Mul(sum, sum), squares)
					effective.Sub(effective, big.NewRat(1, 1))
					effective.Quo(effective, big.NewRat(9, 1))
					if effective.Cmp(big.NewRat(1, 1)) > 0 {
						effective.SetInt64(1)
					}
					effective.Mul(effective, big.NewRat(1, 2))
					effective.Mul(effective, new(big.Rat).SetInt(sum))
					bonus.Quo(effective.Num(), effective.Denom())
				}
				capacity := cc.GetGridAttribute(diversityCapacityKey(id))
				summary := cc.GetOwnerReactorDiversity(id).GetState()
				require.Equal(t, sum.String(), summary.Power)
				require.Equal(t, squares.String(), summary.SquaredPower)
				require.Equal(t, bonus.Uint64(), summary.AppliedBonus)
				require.Equal(t, base+bonus.Uint64(), capacity)
			}
		}
		for i := 0; i+3 < len(input); i += 4 {
			owner, reactor, address := int(input[i]%2), int(input[i+1]%9), int(input[i+2]%3)
			power := uint64(input[i+3]) * uint64(input[i+2])
			old := powers[owner][reactor][address]
			powers[owner][reactor][address] = power
			destination := fmt.Sprintf("3-%d", reactor+1)
			if reactor == 8 {
				destination = "" // helper takes a reactor ID, empty for other sources
			}
			cc.ApplyOwnerInfusionCapacityDelta(owners[owner], destination, old, power)
			if input[i]&4 != 0 {
				check()
			}
			if input[i]&8 != 0 {
				cc.CommitAll()
				cc = k.NewCurrentContext(ctx)
				check()
			}
		}
		check()
		cc.CommitAll()
		cc = k.NewCurrentContext(ctx)
		check()
		for owner, id := range owners {
			state, found := k.GetOwnerReactorDiversity(ctx, id)
			var total uint64
			for _, addresses := range powers[owner][:8] {
				for _, power := range addresses {
					total += power
				}
			}
			require.Equal(t, total != 0, found)
			if found {
				require.Equal(t, id, state.OwnerId)
				require.Equal(t, fmt.Sprint(total), state.Power)
			}
		}
	})
}
