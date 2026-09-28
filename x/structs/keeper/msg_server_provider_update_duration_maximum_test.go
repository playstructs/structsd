package keeper_test

import "testing"

func TestMsgProviderUpdateDurationMaximum(t *testing.T) {
	testProviderBoundUpdate(t, true, true)
}
