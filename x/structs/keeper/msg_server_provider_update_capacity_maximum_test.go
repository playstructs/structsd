package keeper_test

import "testing"

func TestMsgProviderUpdateCapacityMaximum(t *testing.T) {
	testProviderBoundUpdate(t, true, false)
}
