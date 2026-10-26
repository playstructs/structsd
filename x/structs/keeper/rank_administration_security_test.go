package keeper_test

import (
 "testing"

 "github.com/stretchr/testify/require"
 keeperlib "structs/x/structs/keeper"
 "structs/x/structs/types"
)

func TestPurchasedRankCannotEscalateToAdministration(t *testing.T) {
 f:=newNonMemberOwnerFixture(t)
 configureRankSales(t,f,false)
 fundRankVault(t,f,f.member,2000)
 _,err:=f.ms.GuildRankPurchase(f.ctx,&types.MsgGuildRankPurchase{Creator:f.member.Creator,GuildId:f.guild.Id,ExpectedRank:5,TargetRank:2,MaxPrice:2000})
 require.NoError(t,err)
 cc:=f.k.NewCurrentContext(f.ctx)
 cc.SetPermissionsGuildRank(cc.GetGuild(f.guild.Id),cc.GetGuild(f.guild.Id),types.PermAll,2)
 cc.CommitAll()
 c:=f.k.NewCurrentContext(f.ctx)
 actor,err:=c.GetSigningPlayer(f.member.Creator);require.NoError(t,err)
 require.NoError(t,c.PermissionCheck(c.GetGuild(f.guild.Id),actor,types.PermTokenTransfer),"configured operational permission remains usable")
 before:=f.k.GetAllPermissionExport(f.ctx)
 rankBefore:=f.k.GetAllGuildRankPermissionExport(f.ctx)
 _,err=f.ms.PermissionGrantOnObject(f.ctx,&types.MsgPermissionGrantOnObject{Creator:f.member.Creator,ObjectId:f.guild.Id,PlayerId:f.member.Id,Permissions:uint64(types.PermAdmin)});require.Error(t,err)
 _,err=f.ms.PermissionSetOnObject(f.ctx,&types.MsgPermissionSetOnObject{Creator:f.member.Creator,ObjectId:f.guild.Id,PlayerId:f.member.Id,Permissions:uint64(types.PermAll)});require.Error(t,err)
 _,err=f.ms.PermissionRevokeOnObject(f.ctx,&types.MsgPermissionRevokeOnObject{Creator:f.member.Creator,ObjectId:f.guild.Id,PlayerId:f.buyer.Id,Permissions:uint64(types.PermAdmin)});require.Error(t,err)
 _,err=f.ms.PermissionGuildRankSet(f.ctx,&types.MsgPermissionGuildRankSet{Creator:f.member.Creator,ObjectId:f.guild.Id,GuildId:f.guild.Id,Permission:uint64(types.PermAll),Rank:100});require.Error(t,err)
 _,err=f.ms.PermissionGuildRankRevoke(f.ctx,&types.MsgPermissionGuildRankRevoke{Creator:f.member.Creator,ObjectId:f.guild.Id,GuildId:f.guild.Id,Permission:uint64(types.PermAll)});require.Error(t,err)
 _,err=f.ms.GuildUpdateOwnerId(f.ctx,&types.MsgGuildUpdateOwnerId{Creator:f.member.Creator,GuildId:f.guild.Id,Owner:f.member.Id});require.Error(t,err)
 for _,rank:=range []uint64{1,100} {
  _,err=f.ms.PlayerUpdateGuildRank(f.ctx,&types.MsgPlayerUpdateGuildRank{Creator:f.member.Creator,GuildId:f.guild.Id,PlayerId:f.member.Id,GuildRank:rank});require.Error(t,err)
 }
 _,err=f.ms.GuildUpdateEntryRank(f.ctx,&types.MsgGuildUpdateEntryRank{Creator:f.member.Creator,GuildId:f.guild.Id,NewEntryRank:2});require.Error(t,err)
 guild,found:=f.k.GetGuild(f.ctx,f.guild.Id);require.True(t,found);require.Equal(t,f.guild.EntryRank,guild.EntryRank)
 require.Equal(t,before,f.k.GetAllPermissionExport(f.ctx))
 require.Equal(t,rankBefore,f.k.GetAllGuildRankPermissionExport(f.ctx))
 // An explicit administrator may grant only capabilities its signing key and
 // object standing already authorize. The resulting grant is independent.
 f.k.SetPermissionsByBytes(f.ctx,keeperlib.GetObjectPermissionIDBytes(f.guild.Id,f.member.Id),types.PermAdmin|types.PermTokenTransfer)
 _,err=f.ms.PermissionGrantOnObject(f.ctx,&types.MsgPermissionGrantOnObject{Creator:f.member.Creator,ObjectId:f.guild.Id,PlayerId:f.seller.Id,Permissions:uint64(types.PermTokenTransfer)});require.NoError(t,err)
}
