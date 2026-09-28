package keeper_test

import "testing"

func TestMsgProviderUpdateDurationMinimum(t *testing.T) {
	testProviderBoundUpdate(t, false, true)
}
