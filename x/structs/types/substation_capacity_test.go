package types_test

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestSubstationUsableCapacity(t *testing.T) {
	values := []uint64{0, 1, 9, 10, 11, 100, 1000, ^uint64(0)}
	rng := rand.New(rand.NewSource(90))
	for i := 0; i < 1000; i++ {
		values = append(values, rng.Uint64())
	}
	for _, incoming := range values {
		want := new(big.Int).SetUint64(incoming)
		want.Mul(want, big.NewInt(9)).Quo(want, big.NewInt(10))
		require.Equal(t, want.Uint64(), types.SubstationUsableCapacity(incoming), "incoming=%d", incoming)
	}
}
