package app_test

import (
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
	"testing"
)

// Protobuf JSON renders empty repeated fields as [], while wire decoding may
// leave them nil. Compare the full protobuf value, including all pending bases.
func requireInfusionEventEqual(t *testing.T, expected, actual types.Infusion, message string) {
	t.Helper()
	require.Equal(t, expected, actual, message)
}

func requireInfusionEventsEqual(t *testing.T, expected, actual map[string]types.Infusion, message string) {
	t.Helper()
	require.Len(t, actual, len(expected), message)
	for id, row := range expected {
		got, found := actual[id]
		require.True(t, found, "%s: missing %s", message, id)
		requireInfusionEventEqual(t, row, got, message)
	}
}
