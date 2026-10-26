package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgAgreementOpen(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")
	agreement, collateral := f.openAgreement(t, 100, 50)
	require.Equal(t, math.NewInt(50000), collateral)
	require.Equal(t, collateral, f.balance(f.collateralAcc))
	require.True(t, f.balance(f.consumerAcc).IsZero())
	require.True(t, f.balance(f.earningsAcc).IsZero())
	require.Equal(t, f.consumer.Id, agreement.Owner, "omitting OwnerId selects the signer's player vault")
	require.Equal(t, f.consumer.Creator, agreement.Creator)
	require.Equal(t, uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight()), agreement.StartBlock)
	require.Equal(t, agreement.StartBlock+50, agreement.EndBlock)
	require.Contains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, agreement.EndBlock), agreement.Id)
	cc := f.k.NewCurrentContext(f.ctx)
	allocation, found := cc.GetAllocation(agreement.AllocationId)
	require.True(t, found)
	require.Equal(t, f.consumer.Id, allocation.GetOwnerId())
	require.Equal(t, uint64(100), allocation.GetPower())
	require.Equal(t, uint64(100), f.agreementLoad())
	require.Equal(t, types.PermAgreementAll, cc.GetPermissions(keeperlib.GetObjectPermissionIDBytes(agreement.Id, f.consumer.Id)))
	require.Equal(t, types.PermAllocationConnection, cc.GetPermissions(keeperlib.GetObjectPermissionIDBytes(allocation.ID(), f.consumer.Id)))
}
