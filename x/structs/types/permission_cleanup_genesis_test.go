package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestPermissionCleanupGenesisValidation(t *testing.T) {
	for _, id := range []string{"4-9", "6-9", "10-9", "11-9"} {
		gs := types.DefaultGenesis()
		gs.AllocationCount, gs.ProviderCount, gs.SubstationCount = 10, 10, 10
		gs.PermissionCleanupQueue = []string{id}
		require.NoError(t, gs.ValidatePermissionCleanupQueue())
		gs.AllocationCount, gs.ProviderCount, gs.SubstationCount = 9, 9, 9
		require.ErrorContains(t, gs.ValidatePermissionCleanupQueue(), "has not been allocated")
		gs.AllocationCount, gs.ProviderCount, gs.SubstationCount = 0, 0, 0
		require.ErrorContains(t, gs.ValidatePermissionCleanupQueue(), "has not been allocated")
		gs.AllocationCount, gs.ProviderCount, gs.SubstationCount = 10, 10, 10
		gs.PermissionCleanupQueue = append(gs.PermissionCleanupQueue, id)
		require.ErrorContains(t, gs.ValidatePermissionCleanupQueue(), "duplicate")
	}
	for _, id := range []string{"", "6-01", "06-1", "6-+1", "6--1", "6-18446744073709551616", "1-1", "5-1", "999-1"} {
		gs := types.DefaultGenesis()
		gs.AllocationCount, gs.ProviderCount, gs.SubstationCount = 10, 10, 10
		gs.PermissionCleanupQueue = []string{id}
		require.Error(t, gs.ValidatePermissionCleanupQueue(), id)
	}
	gs := types.DefaultGenesis()
	gs.AllocationCount, gs.ProviderCount, gs.SubstationCount = 10, 10, 10
	gs.AllocationList = []types.Allocation{{Id: "6-1"}}
	gs.AgreementList = []types.Agreement{{Id: "11-1"}}
	gs.ProviderList = []types.Provider{{Id: "10-1"}}
	gs.SubstationList = []types.Substation{{Id: "4-1"}}
	for _, id := range []string{"4-1", "6-1", "10-1", "11-1"} {
		gs.PermissionCleanupQueue = []string{id}
		require.ErrorContains(t, gs.ValidatePermissionCleanupQueue(), "live object")
	}
}
