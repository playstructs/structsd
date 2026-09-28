package app_test

import (
 "testing"

 sdk "github.com/cosmos/cosmos-sdk/types"
 "github.com/stretchr/testify/require"
 v22 "structs/app/upgrades/v0_22_0"
 keeperlib "structs/x/structs/keeper"
 "structs/x/structs/types"
)

func TestUpgradeDoesNotActivateInertGuildTreasuryGrants(t *testing.T) {
 bApp,ctx:=setupJailGateApp(t)
 _,_,owner:=newTransferPlayer(t,bApp,ctx)
 signer,_,member:=newTransferPlayer(t,bApp,ctx)
 k:=bApp.StructsKeeper
 guild:=types.CreateEmptyGuild();guild.Id="0-995";guild.Index=995;guild.Owner=owner;k.SetGuild(ctx,guild)
 id:=keeperlib.GetObjectPermissionIDBytes(guild.Id,member)
 newBits:=types.PermAssetsAll|types.PermSourceAllocation|types.PermAllocationConnection
 k.SetPermissionsByBytes(ctx,id,newBits|types.PermGuildEndpointUpdate)
 k.SetGuildRankPermissionStoreOnly(ctx,guild.Id,guild.Id,newBits|types.PermGuildEndpointUpdate,3)
 // The same direct asset permissions on a player's own wallet remain valid.
 playerID:=keeperlib.GetObjectPermissionIDBytes(member,member)
 k.SetPermissionsByBytes(ctx,playerID,newBits)
 ctx=ctx.WithEventManager(sdk.NewEventManager())
 require.NoError(t,v22.MigrateGuildEconomicPermissions(ctx,delegationUpgradeKeepers(bApp)))
 require.Equal(t,types.PermGuildEndpointUpdate,k.GetPermissionsByBytes(ctx,id))
 require.Equal(t,newBits,k.GetPermissionsByBytes(ctx,playerID))
 c:=k.NewCurrentContext(ctx)
 _,found:=c.GetPermissionsGuildRank(c.GetGuild(guild.Id),c.GetGuild(guild.Id),newBits);require.False(t,found)
 rank,found:=c.GetPermissionsGuildRank(c.GetGuild(guild.Id),c.GetGuild(guild.Id),types.PermGuildEndpointUpdate);require.True(t,found);require.Equal(t,uint64(3),rank)
 actor,err:=c.GetSigningPlayer(signer.String());require.NoError(t,err)
 require.Error(t,c.PermissionCheck(c.GetGuild(guild.Id),actor,types.PermTokenTransfer))
 var permissionEvent bool
 for _,event:=range ctx.EventManager().ABCIEvents(){
  if event.Type!="structs.structs.EventPermission" {continue}
  parsed,err:=sdk.ParseTypedEvent(event);require.NoError(t,err)
  row:=parsed.(*types.EventPermission).PermissionRecord
  if row.PermissionId==string(id){require.Equal(t,uint64(types.PermGuildEndpointUpdate),row.Value);permissionEvent=true}
 }
 require.True(t,permissionEvent,"indexer receives the resulting direct grant")
 before:=len(ctx.EventManager().Events())
 require.NoError(t,v22.MigrateGuildEconomicPermissions(ctx,delegationUpgradeKeepers(bApp)))
 require.Len(t,ctx.EventManager().Events(),before,"rerun changes no rights or events")
}
