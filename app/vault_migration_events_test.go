package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func migrationEvents[T any](t testing.TB, ctx sdk.Context, name string) []T {
	t.Helper()
	events := []T{}
	for _, event := range ctx.EventManager().ABCIEvents() {
		if event.Type != "structs.structs."+name { continue }
		decoded, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		typed, ok := decoded.(T)
		require.True(t, ok)
		events = append(events, typed)
	}
	return events
}
