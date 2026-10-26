package keeper_test

import (
 "testing"

 storetypes "cosmossdk.io/store/types"
 sdk "github.com/cosmos/cosmos-sdk/types"
 "github.com/stretchr/testify/require"
 keeperlib "structs/x/structs/keeper"
 "structs/x/structs/types"
)

func TestOpenAllocationContributionsCannotExhaustVictimReadiness(t *testing.T) {
 chain := buildCascadeChain(t,1)
 attackerAddress := sdk.AccAddress("allocation-donor-0001")
 attacker := testAppendPlayer(chain.k,chain.ctx,types.Player{Creator:attackerAddress.String(),PrimaryAddress:attackerAddress.String()})
 chain.k.SetGridAttribute(chain.ctx,keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity,attacker.Id),10000)
 cc:=chain.k.NewCurrentContext(chain.ctx)
 cc.GetPlayer(chain.owner.Id).MigrateSubstation(chain.substationIds[0])
 cc.CommitAll()
 server:=keeperlib.NewMsgServerImpl(chain.k)
 measure:=func() uint64 {
  ctx:=chain.ctx.WithGasMeter(storetypes.NewGasMeter(20_000_000))
  c:=chain.k.NewCurrentContext(ctx)
  require.True(t,c.GetPlayer(chain.owner.Id).IsOnline())
  require.Positive(t,c.GetSubstation(chain.substationIds[0]).GetAvailableCapacity())
  return ctx.GasMeter().GasConsumed()
 }
 var small uint64
 for i:=0;i<2050;i++ {
  created,err:=server.AllocationCreate(chain.ctx,&types.MsgAllocationCreate{Creator:attacker.Creator,Controller:attacker.Id,SourceObjectId:attacker.Id,AllocationType:types.AllocationType_dynamic,Power:1})
  require.NoError(t,err)
  _,err=server.SubstationAllocationConnect(chain.ctx,&types.MsgSubstationAllocationConnect{Creator:attacker.Creator,AllocationId:created.AllocationId,DestinationId:chain.substationIds[0]})
  require.NoError(t,err,"donor does not need authority on the destination")
  if i==9 {small=measure()}
 }
 require.Equal(t,small,measure(),"victim readiness cost must not grow with donated allocations")
 require.Equal(t,types.Permissionless,chain.k.GetPermissionsByBytes(chain.ctx,keeperlib.GetObjectPermissionIDBytes(chain.substationIds[0],attacker.Id)))
}
