package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestGenesisAllocationTypesAreClosed(t *testing.T) {
	for _, value := range []int32{-2147483648, -1, 0, 1, 2, 3, 4, 500, 2147483647} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			genesis := types.DefaultGenesis()
			genesis.AllocationList = []types.Allocation{{Id: "6-1", Index: 1, Type: types.AllocationType(value)}}
			err := genesis.Validate()
			if value >= 0 && value <= 3 {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, types.ErrAllocationCreate)
			}
		})
	}
}
