package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Guild ownership without membership.
 *
 * A transfer moves guild.Owner and the PermGuildAll row and touches neither
 * player's GuildId, so selling a guild inherently produces an owner who is not a
 * member. PermissionCheck's owner shortcut means such an owner still passes every
 * authorization check on their guild, so this was never an authorization gap —
 * but four handlers resolved the guild solely from the signer's membership, and
 * so could not be *addressed* to a guild the signer was not in.
 *
 * Each of them now takes a guildId, with membership as the fallback. The fallback
 * cases below matter as much as the new ones: an existing client sends no guildId
 * and must keep working unchanged.
 *
 * Kick, invite and request-approve are the leftover membership gates: rank
 * authority is members-only in the named guild, and at bypass level member the
 * owner counts as able to staff the guild without joining it.
 */

// nonMemberOwnerFixture is a guild whose owner is not a member of it, produced
// the way the real thing is: a sale.
type nonMemberOwnerFixture struct {
	t   *testing.T
	k   keeperlib.Keeper
	ms  types.MsgServer
	ctx sdk.Context

	guild  types.Guild
	seller types.Player
	buyer  types.Player
	member types.Player
}

func newNonMemberOwnerFixture(t *testing.T) *nonMemberOwnerFixture {
	t.Helper()

	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	sellerAcc := sdk.AccAddress("guildseller1234567890123456789012345")
	seller := testAppendPlayer(k, ctx, types.Player{
		Creator:        sellerAcc.String(),
		PrimaryAddress: sellerAcc.String(),
	})

	reactor := k.AppendReactor(ctx, types.Reactor{
		Validator:  sdk.ValAddress(sellerAcc.Bytes()).String(),
		RawAddress: sdk.ValAddress(sellerAcc.Bytes()).Bytes(),
		Owner:      seller.Id,
	})

	guild := k.AppendGuild(ctx, "owner-test", "", reactor, seller, "")
	seller.GuildId = guild.Id
	seller.GuildRank = 1
	k.SetPlayer(ctx, seller)

	buyerAcc := sdk.AccAddress("guildbuyer1234567890123456789012345")
	buyer := testAppendPlayer(k, ctx, types.Player{
		Creator:        buyerAcc.String(),
		PrimaryAddress: buyerAcc.String(),
	})

	// A rank-and-file member, who is who the rank handler acts on.
	memberAcc := sdk.AccAddress("guildmember123456789012345678901234")
	member := testAppendPlayer(k, ctx, types.Player{
		Creator:        memberAcc.String(),
		PrimaryAddress: memberAcc.String(),
	})
	member.GuildId = guild.Id
	member.GuildRank = 5
	k.SetPlayer(ctx, member)

	_, err := ms.GuildUpdateOwnerId(ctx, &types.MsgGuildUpdateOwnerId{
		Creator: seller.Creator,
		GuildId: guild.Id,
		Owner:   buyer.Id,
	})
	require.NoError(t, err)

	storedBuyer, found := k.GetPlayer(ctx, buyer.Id)
	require.True(t, found)
	require.Empty(t, storedBuyer.GuildId,
		"the fixture is only meaningful if the buyer is not a member of what they bought")

	return &nonMemberOwnerFixture{
		t: t, k: k, ms: ms, ctx: ctx,
		guild: guild, seller: seller, buyer: buyer, member: member,
	}
}

// fundAlpha gives a player the ualpha a mint needs as collateral.
func (f *nonMemberOwnerFixture) fundAlpha(player types.Player, amount int64) {
	f.t.Helper()

	acc, err := sdk.AccAddressFromBech32(player.PrimaryAddress)
	require.NoError(f.t, err)

	coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(amount)))
	require.NoError(f.t, f.k.BankKeeper().MintCoins(f.ctx, types.ModuleName, coins))
	require.NoError(f.t, f.k.BankKeeper().SendCoinsFromModuleToAccount(f.ctx, types.ModuleName, acc, coins))
}

func (f *nonMemberOwnerFixture) tokenBalance(player types.Player) math.Int {
	f.t.Helper()

	acc, err := sdk.AccAddressFromBech32(player.PrimaryAddress)
	require.NoError(f.t, err)

	guild, found := f.k.GetGuild(f.ctx, f.guild.Id)
	require.True(f.t, found)

	return f.k.BankKeeper().SpendableCoin(f.ctx, acc, "uguild."+guild.Id).Amount
}

// TestNonMemberOwnerMintsAndBurns covers the two bank handlers, which are the
// pair where being unable to name the guild would mean being unable to run its
// currency at all.
func TestNonMemberOwnerMintsAndBurns(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	f.fundAlpha(f.buyer, 1000)

	_, err := f.ms.GuildBankMint(f.ctx, &types.MsgGuildBankMint{
		Creator:     f.buyer.Creator,
		GuildId:     f.guild.Id,
		AmountAlpha: 100,
		AmountToken: 10,
	})
	require.NoError(t, err, "an owner must be able to mint their guild's token without joining it")
	require.Equal(t, math.NewInt(10), f.tokenBalance(f.buyer))

	_, err = f.ms.GuildBankConfiscateAndBurn(f.ctx, &types.MsgGuildBankConfiscateAndBurn{
		Creator:     f.buyer.Creator,
		GuildId:     f.guild.Id,
		Address:     f.buyer.PrimaryAddress,
		AmountToken: 4,
	})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(6), f.tokenBalance(f.buyer))
}

// TestNonMemberOwnerSetsEntryRank is the plain setter case, and it also pins the
// rank ceiling as a members-only rule.
func TestNonMemberOwnerSetsEntryRank(t *testing.T) {
	f := newNonMemberOwnerFixture(t)

	_, err := f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      f.buyer.Creator,
		GuildId:      f.guild.Id,
		NewEntryRank: 7,
	})
	require.NoError(t, err)

	guild, _ := f.k.GetGuild(f.ctx, f.guild.Id)
	require.Equal(t, uint64(7), guild.EntryRank)

	/* The ceiling is "no better than your own rank", which is a fact about a
	 * caller's standing inside the guild being edited. A guildless owner has no
	 * rank at all, and an owner who is a member somewhere else has one that says
	 * nothing about this guild, so applying it to either would be reading an
	 * unrelated number.
	 */
	_, err = f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      f.buyer.Creator,
		GuildId:      f.guild.Id,
		NewEntryRank: 1,
	})
	require.NoError(t, err, "a non-member owner has no rank of their own to be limited by")
}

// TestNonMemberOwnerSetsMemberRank is the one handler whose rule had to be
// restated: "the target shares my guild" becomes "the target is in the guild
// named", which is the same test whenever the caller is a member.
func TestNonMemberOwnerSetsMemberRank(t *testing.T) {
	f := newNonMemberOwnerFixture(t)

	_, err := f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      f.buyer.Creator,
		GuildId:      f.guild.Id,
		NewEntryRank: 9,
	})
	require.NoError(t, err)

	_, err = f.ms.PlayerUpdateGuildRank(f.ctx, &types.MsgPlayerUpdateGuildRank{
		Creator:   f.buyer.Creator,
		GuildId:   f.guild.Id,
		PlayerId:  f.member.Id,
		GuildRank: 3,
	})
	require.NoError(t, err)

	member, _ := f.k.GetPlayer(f.ctx, f.member.Id)
	require.Equal(t, uint64(3), member.GuildRank)

	// A player in some other guild is still out of reach, which is the check
	// that survived the restatement.
	outsiderAcc := sdk.AccAddress("guildoutsider1234567890123456789012")
	outsider := testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        outsiderAcc.String(),
		PrimaryAddress: outsiderAcc.String(),
	})

	_, err = f.ms.PlayerUpdateGuildRank(f.ctx, &types.MsgPlayerUpdateGuildRank{
		Creator:   f.buyer.Creator,
		GuildId:   f.guild.Id,
		PlayerId:  outsider.Id,
		GuildRank: 3,
	})
	require.Error(t, err, "owning a guild is not authority over players outside it")
}

/* TestNonMemberOwnerNeedsPermission is the check the new field must not have
 * bypassed.
 *
 * A message-supplied guild id is a target, so naming one buys nothing on its own:
 * what authorizes each of these is the same permission check as before, which the
 * owner passes through PermissionCheck's owner shortcut and a stranger does not.
 */
func TestNonMemberOwnerNeedsPermission(t *testing.T) {
	f := newNonMemberOwnerFixture(t)

	strangerAcc := sdk.AccAddress("guildstranger123456789012345678901")
	stranger := testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        strangerAcc.String(),
		PrimaryAddress: strangerAcc.String(),
	})
	f.fundAlpha(stranger, 1000)

	_, err := f.ms.GuildBankMint(f.ctx, &types.MsgGuildBankMint{
		Creator:     stranger.Creator,
		GuildId:     f.guild.Id,
		AmountAlpha: 100,
		AmountToken: 10,
	})
	require.Error(t, err, "naming a guild must not stand in for holding rights on it")

	_, err = f.ms.GuildBankConfiscateAndBurn(f.ctx, &types.MsgGuildBankConfiscateAndBurn{
		Creator:     stranger.Creator,
		GuildId:     f.guild.Id,
		Address:     f.buyer.PrimaryAddress,
		AmountToken: 1,
	})
	require.Error(t, err)

	_, err = f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      stranger.Creator,
		GuildId:      f.guild.Id,
		NewEntryRank: 2,
	})
	require.Error(t, err)

	_, err = f.ms.PlayerUpdateGuildRank(f.ctx, &types.MsgPlayerUpdateGuildRank{
		Creator:   stranger.Creator,
		GuildId:   f.guild.Id,
		PlayerId:  f.member.Id,
		GuildRank: 2,
	})
	require.Error(t, err, "a stranger with no rank in the guild has only PermAdmin to fall back on")

	// A guild id that does not resolve must be refused rather than allocated:
	// cc.GetGuild is a cache allocator, and a phantom reads as a zero value whose
	// owner is the empty string.
	_, err = f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      f.buyer.Creator,
		GuildId:      "5-9999",
		NewEntryRank: 2,
	})
	require.Error(t, err)

	_, err = f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      f.buyer.Creator,
		GuildId:      f.guild.Id,
		NewEntryRank: 0,
	})
	require.Error(t, err, "rank zero would outrank every ordinary member")
}

/* TestGuildHandlerMembershipFallback is the compatibility half: a client that
 * predates the new field sends nothing and must be unaffected.
 */
func TestGuildHandlerMembershipFallback(t *testing.T) {
	f := newNonMemberOwnerFixture(t)

	// Grant the member the bits a permissioned officer would hold, so the
	// fallback is exercised by somebody other than an owner.
	testPermissionAdd(f.k, f.ctx, keeperlib.GetObjectPermissionIDBytes(f.guild.Id, f.member.Id), types.PermGuildTokenMint)
	testPermissionAdd(f.k, f.ctx, keeperlib.GetObjectPermissionIDBytes(f.guild.Id, f.member.Id), types.PermUpdate)
	f.fundAlpha(f.member, 1000)

	_, err := f.ms.GuildBankMint(f.ctx, &types.MsgGuildBankMint{
		Creator:     f.member.Creator,
		AmountAlpha: 100,
		AmountToken: 10,
	})
	require.NoError(t, err, "an omitted guild id must still mean the signer's own guild")
	require.Equal(t, math.NewInt(10), f.tokenBalance(f.member))

	_, err = f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      f.member.Creator,
		NewEntryRank: 6,
	})
	require.NoError(t, err)

	guild, _ := f.k.GetGuild(f.ctx, f.guild.Id)
	require.Equal(t, uint64(6), guild.EntryRank)

	// And the members-only ceiling still binds a member: rank 5 cannot set an
	// entry rank better than 5.
	_, err = f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      f.member.Creator,
		NewEntryRank: 2,
	})
	require.Error(t, err, "a member must not set an entry rank better than their own")

	// A guildless signer naming nothing has no guild to fall back to.
	strangerAcc := sdk.AccAddress("guildnobody12345678901234567890123")
	stranger := testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        strangerAcc.String(),
		PrimaryAddress: strangerAcc.String(),
	})

	_, err = f.ms.GuildUpdateEntryRank(f.ctx, &types.MsgGuildUpdateEntryRank{
		Creator:      stranger.Creator,
		NewEntryRank: 6,
	})
	require.Error(t, err)
}

func (f *nonMemberOwnerFixture) setBypass(invite, request types.GuildJoinBypassLevel) {
	f.t.Helper()

	guild, found := f.k.GetGuild(f.ctx, f.guild.Id)
	require.True(f.t, found)
	guild.JoinInfusionMinimumBypassByInvite = invite
	guild.JoinInfusionMinimumBypassByRequest = request
	f.k.SetGuild(f.ctx, guild)
	f.guild = guild
}

func (f *nonMemberOwnerFixture) guildlessPlayer(seed string) types.Player {
	f.t.Helper()

	acc := sdk.AccAddress(seed)
	return testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        acc.String(),
		PrimaryAddress: acc.String(),
	})
}

// TestNonMemberOwnerKicksMember is the kick half of the restated rank rule.
// CanKickMembers already passes the owner; the outrank test must not then
// read a guildless owner's zero as a rank in this guild.
func TestNonMemberOwnerKicksMember(t *testing.T) {
	f := newNonMemberOwnerFixture(t)

	unranked := f.guildlessPlayer("guildrankzero123456789012345678901")
	unranked.GuildId = f.guild.Id
	unranked.GuildRank = 0
	f.k.SetPlayer(f.ctx, unranked)

	_, err := f.ms.GuildMembershipKick(f.ctx, &types.MsgGuildMembershipKick{
		Creator:  f.buyer.Creator,
		GuildId:  f.guild.Id,
		PlayerId: unranked.Id,
	})
	require.NoError(t, err, "a guildless owner must not be limited by a rank they do not hold here")

	kicked, found := f.k.GetPlayer(f.ctx, unranked.Id)
	require.True(t, found)
	require.Empty(t, kicked.GuildId)

	_, err = f.ms.GuildMembershipKick(f.ctx, &types.MsgGuildMembershipKick{
		Creator:  f.buyer.Creator,
		GuildId:  f.guild.Id,
		PlayerId: f.member.Id,
	})
	require.NoError(t, err)

	member, found := f.k.GetPlayer(f.ctx, f.member.Id)
	require.True(t, found)
	require.Empty(t, member.GuildId)
}

// TestNonMemberOwnerKicksWhileMemberElsewhere pins the other half: a rank in
// some other guild says nothing about who this owner may remove.
func TestNonMemberOwnerKicksWhileMemberElsewhere(t *testing.T) {
	f := newNonMemberOwnerFixture(t)

	buyer, found := f.k.GetPlayer(f.ctx, f.buyer.Id)
	require.True(t, found)
	buyer.GuildId = "5-999"
	buyer.GuildRank = 5
	f.k.SetPlayer(f.ctx, buyer)

	_, err := f.ms.GuildMembershipKick(f.ctx, &types.MsgGuildMembershipKick{
		Creator:  f.buyer.Creator,
		GuildId:  f.guild.Id,
		PlayerId: f.member.Id,
	})
	require.NoError(t, err, "a rank in another guild must not decide a kick here")

	member, found := f.k.GetPlayer(f.ctx, f.member.Id)
	require.True(t, found)
	require.Empty(t, member.GuildId)
}

// TestNonMemberOwnerInvitesAndApprovesAtMemberBypass is the remaining
// "you must be in it" gate: at member, invite and approve used to demand
// Player.GuildId equality even of the owner. member still means any member
// may invite; the owner now counts as well. permissioned is unchanged —
// the owner already passed through PermissionCheck, and an ordinary member
// still does not.
func TestNonMemberOwnerInvitesAndApprovesAtMemberBypass(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	f.setBypass(types.GuildJoinBypassLevel_member, types.GuildJoinBypassLevel_member)

	invitee := f.guildlessPlayer("guildinvitee12345678901234567890123")
	_, err := f.ms.GuildMembershipInvite(f.ctx, &types.MsgGuildMembershipInvite{
		Creator:  f.buyer.Creator,
		GuildId:  f.guild.Id,
		PlayerId: invitee.Id,
	})
	require.NoError(t, err, "an owner must be able to invite without joining")

	app, found := f.k.GetGuildMembershipApplication(f.ctx, f.guild.Id, invitee.Id)
	require.True(t, found)
	require.Equal(t, types.RegistrationStatus_proposed, app.RegistrationStatus)

	// Ordinary members still invite at this level; that is the intra-guild
	// policy, not the owner exception.
	colleague := f.guildlessPlayer("guildcolleague12345678901234567890")
	_, err = f.ms.GuildMembershipInvite(f.ctx, &types.MsgGuildMembershipInvite{
		Creator:  f.member.Creator,
		GuildId:  f.guild.Id,
		PlayerId: colleague.Id,
	})
	require.NoError(t, err, "any member may still invite at bypass level member")

	stranger := f.guildlessPlayer("guildstrangerinv123456789012345678")
	_, err = f.ms.GuildMembershipInvite(f.ctx, &types.MsgGuildMembershipInvite{
		Creator:  stranger.Creator,
		GuildId:  f.guild.Id,
		PlayerId: colleague.Id,
	})
	require.Error(t, err, "naming the guild is not membership")
	require.ErrorIs(t, err, types.ErrGuildMembership)

	requester := f.guildlessPlayer("guildrequester12345678901234567890")
	_, err = f.ms.GuildMembershipRequest(f.ctx, &types.MsgGuildMembershipRequest{
		Creator:  requester.Creator,
		GuildId:  f.guild.Id,
		PlayerId: requester.Id,
	})
	require.NoError(t, err)

	_, err = f.ms.GuildMembershipRequestApprove(f.ctx, &types.MsgGuildMembershipRequestApprove{
		Creator:  f.buyer.Creator,
		GuildId:  f.guild.Id,
		PlayerId: requester.Id,
	})
	require.NoError(t, err, "an owner must be able to approve a request without joining")

	joined, found := f.k.GetPlayer(f.ctx, requester.Id)
	require.True(t, found)
	require.Equal(t, f.guild.Id, joined.GuildId)
}

func TestNonMemberOwnerInvitesAtPermissionedBypass(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	f.setBypass(types.GuildJoinBypassLevel_permissioned, types.GuildJoinBypassLevel_permissioned)

	invitee := f.guildlessPlayer("guildperminvitee123456789012345678")
	_, err := f.ms.GuildMembershipInvite(f.ctx, &types.MsgGuildMembershipInvite{
		Creator:  f.buyer.Creator,
		GuildId:  f.guild.Id,
		PlayerId: invitee.Id,
	})
	require.NoError(t, err, "permissioned still passes the owner through PermissionCheck")

	colleague := f.guildlessPlayer("guildpermcolleague1234567890123456")
	_, err = f.ms.GuildMembershipInvite(f.ctx, &types.MsgGuildMembershipInvite{
		Creator:  f.member.Creator,
		GuildId:  f.guild.Id,
		PlayerId: colleague.Id,
	})
	require.Error(t, err, "an ordinary member still needs PermGuildMembership at permissioned")
}
