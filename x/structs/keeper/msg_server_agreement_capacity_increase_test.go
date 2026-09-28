package keeper_test

import "testing"

func TestMsgAgreementCapacityIncrease(t *testing.T) {
	testAgreementMutationAuthority(t, "increase")
}
