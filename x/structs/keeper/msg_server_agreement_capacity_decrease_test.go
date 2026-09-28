package keeper_test

import "testing"

func TestMsgAgreementCapacityDecrease(t *testing.T) {
	testAgreementMutationAuthority(t, "decrease")
}
