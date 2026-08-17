package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// addressMoveFixture is a player holding a delegation on some address other
// than the one the move is going to end at, with the infusion behind it built
// through the normal staking path.
//
// It exists to cover a regression the mock cannot reach on its own: the three
// address-move handlers call RemoveDelegation, which fires
// BeforeDelegationRemoved and zeroes the source infusion, then SetDelegation,
// which is a bare store write and fires nothing. Left to the hooks the player's
// capacity is destroyed rather than relocated. The handlers reconcile both
// sides explicitly, which is what makes the repair visible here even though
// MockStakingKeeper fires no hooks at all.
type addressMoveFixture struct {
	k       keeperlib.Keeper
	ms      types.MsgServer
	ctx     context.Context
	sdkCtx  sdk.Context
	mock    *keepertest.MockStakingKeeper
	distr   *keepertest.MockDistributionKeeper
	player  types.Player
	valAddr sdk.ValAddress
	reactor types.Reactor
}

func (f addressMoveFixture) playerCapacity() uint64 {
	return f.k.GetGridAttribute(f.sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.player.Id))
}

func (f addressMoveFixture) infusionAt(address string) (types.Infusion, bool) {
	return f.k.GetInfusion(f.sdkCtx, f.reactor.Id, address)
}

// setupAddressMove seeds the player, the reactor, and a delegation held by
// holderAcc. seed must be unique per test so the generated addresses do not
// collide across the package.
func setupAddressMove(t *testing.T, seed string, primaryAcc sdk.AccAddress, holderAcc sdk.AccAddress, tokens int64) addressMoveFixture {
	t.Helper()

	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        primaryAcc.String(),
		PrimaryAddress: primaryAcc.String(),
	})

	// The reactor's validator is somebody else's, so the move is a pure
	// delegator-side operation.
	valAddr := sdk.ValAddress(fmt.Sprintf("%-36s", seed+"val")[:36])
	testAddValidator(k, valAddr, math.NewInt(tokens))

	defaultCommission, err := math.LegacyNewDecFromStr("0.04")
	require.NoError(t, err)

	reactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:         valAddr.String(),
		RawAddress:        valAddr.Bytes(),
		DefaultCommission: defaultCommission,
	})

	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{
		DelegatorAddress: holderAcc.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(tokens)),
	}))

	// A delegation written by staking's own Delegate always carries the
	// distribution starting info that prices its rewards, and the transfer
	// refuses to move one that does not, so the fixture has to supply it.
	distr := k.DistributionKeeper().(*keepertest.MockDistributionKeeper)
	distr.SeedStartingInfo(valAddr, holderAcc)

	return addressMoveFixture{
		k:       k,
		ms:      ms,
		ctx:     ctx,
		sdkCtx:  sdkCtx,
		mock:    mock,
		distr:   distr,
		player:  player,
		valAddr: valAddr,
		reactor: reactor,
	}
}

// assertDelegationMoved is the shared set of assertions for all three handlers:
// the stake and the capacity behind it end up at the destination address, and
// the player is no worse off than before.
func assertDelegationMoved(t *testing.T, f addressMoveFixture, from sdk.AccAddress, to sdk.AccAddress) {
	t.Helper()

	require.Equal(t, uint64(960), f.playerCapacity(),
		"the player's capacity must follow the stake, not be destroyed by the move")

	destination, found := f.infusionAt(to.String())
	require.True(t, found, "the destination address must hold the infusion")
	require.Equal(t, uint64(1000), destination.Fuel)
	require.Equal(t, f.player.Id, destination.PlayerId)

	source, found := f.infusionAt(from.String())
	if found {
		require.Equal(t, uint64(0), source.Fuel, "the source address must keep no stake")
		require.Equal(t, uint64(0), source.Power)
	}

	require.Equal(t, uint64(1000), f.k.GetGridAttribute(f.sdkCtx,
		keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, f.reactor.Id)),
		"the reactor's fuel is unchanged; the stake moved address, not validator")
	require.Equal(t, uint64(40), f.k.GetGridAttribute(f.sdkCtx,
		keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.reactor.Id)),
		"and so is the commission it earns on that stake")

	_, delegationErr := f.mock.GetDelegation(f.ctx, to, f.valAddr)
	require.NoError(t, delegationErr, "staking must show the delegation at the destination")
}

// TestAddressRevokePreservesInfusionCapacity covers revoking a secondary address
// that holds delegations. Before the repair the revoke destroyed the player's
// energy outright.
func TestAddressRevokePreservesInfusionCapacity(t *testing.T) {
	primaryAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "revokeinfusionprimary")[:36])
	secondaryAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "revokeinfusionsecondary")[:36])

	f := setupAddressMove(t, "revokeinfusion", primaryAcc, secondaryAcc, 1000)

	require.NoError(t, f.k.SetPlayerIndexForAddress(f.ctx, secondaryAcc.String(), f.player.Index))
	testPermissionAdd(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(secondaryAcc.String()), types.PermAll)

	f.k.ReactorUpdatePlayerInfusion(f.ctx, secondaryAcc, f.valAddr)
	require.Equal(t, uint64(960), f.playerCapacity(), "precondition: the secondary address funds the player")

	_, err := f.ms.AddressRevoke(f.ctx, &types.MsgAddressRevoke{
		Creator: primaryAcc.String(),
		Address: secondaryAcc.String(),
	})
	require.NoError(t, err)

	assertDelegationMoved(t, f, secondaryAcc, primaryAcc)
	require.Equal(t, uint64(0), f.k.GetPlayerIndexFromAddress(f.sdkCtx, secondaryAcc.String()),
		"the address is still revoked")
}

// TestPlayerUpdatePrimaryAddressPreservesInfusionCapacity covers the same move
// in the other direction: the delegation starts on the old primary address and
// has to arrive on the new one.
func TestPlayerUpdatePrimaryAddressPreservesInfusionCapacity(t *testing.T) {
	oldPrimaryAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "updateprimaryinfusionold")[:36])
	newPrimaryAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "updateprimaryinfusionnew")[:36])

	f := setupAddressMove(t, "updateprimaryinfusion", oldPrimaryAcc, oldPrimaryAcc, 1000)

	require.NoError(t, f.k.SetPlayerIndexForAddress(f.ctx, newPrimaryAcc.String(), f.player.Index))
	testPermissionAdd(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(newPrimaryAcc.String()), types.PermAll)

	f.k.ReactorUpdatePlayerInfusion(f.ctx, oldPrimaryAcc, f.valAddr)
	require.Equal(t, uint64(960), f.playerCapacity())

	_, err := f.ms.PlayerUpdatePrimaryAddress(f.ctx, &types.MsgPlayerUpdatePrimaryAddress{
		Creator:        oldPrimaryAcc.String(),
		PrimaryAddress: newPrimaryAcc.String(),
	})
	require.NoError(t, err)

	assertDelegationMoved(t, f, oldPrimaryAcc, newPrimaryAcc)
}

// TestAddressRegisterSweepPreservesInfusionCapacity covers the third mover,
// AddressRegister, which sweeps an incoming address's delegations onto the
// player's primary address. This one exercises MoveDelegationsToAddress rather
// than the handler: AddressRegister's key proof compares msg.Address against
// types.PubKeyToBech32, which hardcodes the "structs" prefix, and only the CLI
// configures that prefix, so under a keeper test's default "cosmos" prefix the
// proof can never pass. All three handlers call this one function, and the other
// two drive it end-to-end above.
//
// Register is also the interesting shape for ownership: the incoming address
// arrives already infused, and the record can still name whoever held it before.
func TestAddressRegisterSweepPreservesInfusionCapacity(t *testing.T) {
	primaryAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "registerinfusionprimary")[:36])
	incomingAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "registerinfusionincoming")[:36])

	f := setupAddressMove(t, "registerinfusion", primaryAcc, incomingAcc, 1000)

	strangerAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "registerinfusionstranger")[:36])
	stranger := testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        strangerAcc.String(),
		PrimaryAddress: strangerAcc.String(),
	})
	testAppendInfusion(f.k, f.ctx, types.Infusion{
		DestinationId:   f.reactor.Id,
		DestinationType: types.ObjectType_reactor,
		Address:         incomingAcc.String(),
		PlayerId:        stranger.Id,
		Fuel:            1000,
		Commission:      math.LegacyZeroDec(),
	})

	cc := f.k.NewCurrentContext(f.sdkCtx)
	// Match AddressRegister exactly: the association exists only in this
	// CurrentContext until CommitAll.
	cc.SetPlayerIndexForAddress(incomingAcc.String(), f.player.Index)
	playerCount := f.k.GetPlayerCount(f.sdkCtx)
	require.NoError(t, f.k.MoveDelegationsToAddress(f.sdkCtx, cc, incomingAcc, primaryAcc.String(), keeperlib.DelegationTransferStrict))
	cc.CommitAll()

	require.Equal(t, playerCount, f.k.GetPlayerCount(f.sdkCtx),
		"reconciling the staged address must not synthesize an orphan player")
	assertDelegationMoved(t, f, incomingAcc, primaryAcc)
	require.Equal(t, uint64(0), f.k.GetGridAttribute(f.sdkCtx,
		keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, stranger.Id)),
		"the stranger the record named never had a claim on this stake")
}

// TestGuildMembershipJoinRefusesReassignedInfusionAddress is the reported attack.
// The former holder of an address names the infusion still bearing their player
// id and tries to redelegate the stake it now describes, which belongs to
// whoever the address was re-registered to.
//
// The stored PlayerId is exactly what the attacker is relying on, so the check
// that refuses them has to be the live one: resolve the address and ask who owns
// it now.
func TestGuildMembershipJoinRefusesReassignedInfusionAddress(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	guild, found := k.GetGuild(sdkCtx, gs.Guild.Id)
	require.True(t, found)
	guild.JoinInfusionMinimum = 500
	k.SetGuild(sdkCtx, guild)

	attackerAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "joinstaleattacker")[:36])
	attacker := testAppendPlayer(k, ctx, types.Player{
		Creator:        attackerAcc.String(),
		PrimaryAddress: attackerAcc.String(),
	})

	// The disputed address. It was the attacker's once; it is the victim's now,
	// which is all the index records.
	disputedAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "joinstaledisputed")[:36])
	victim := testAppendPlayer(k, ctx, types.Player{
		Creator:        disputedAcc.String(),
		PrimaryAddress: disputedAcc.String(),
	})

	sourceValAddr := sdk.ValAddress(fmt.Sprintf("%-36s", "joinstalesourceval")[:36])
	testAddValidator(k, sourceValAddr, math.NewInt(1000))
	sourceReactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:         sourceValAddr.String(),
		RawAddress:        sourceValAddr.Bytes(),
		DefaultCommission: math.LegacyZeroDec(),
	})

	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{
		DelegatorAddress: disputedAcc.String(),
		ValidatorAddress: sourceValAddr.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(1000)),
	}))

	// The stale record: the victim's address, still carrying the attacker's id.
	testAppendInfusion(k, ctx, types.Infusion{
		DestinationId:   sourceReactor.Id,
		DestinationType: types.ObjectType_reactor,
		Address:         disputedAcc.String(),
		PlayerId:        attacker.Id,
		Fuel:            1000,
		Commission:      math.LegacyZeroDec(),
	})
	infusionId := sourceReactor.Id + "-" + disputedAcc.String()

	_, err := ms.GuildMembershipJoin(ctx, &types.MsgGuildMembershipJoin{
		Creator:    attacker.Creator,
		GuildId:    gs.Guild.Id,
		PlayerId:   attacker.Id,
		InfusionId: []string{infusionId},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "infusion_address_ownership")

	// Nothing may have moved. The victim's stake is still where they put it.
	delegation, delegationErr := mock.GetDelegation(ctx, disputedAcc, sourceValAddr)
	require.NoError(t, delegationErr, "the victim's delegation must be untouched")
	require.Equal(t, math.LegacyNewDecFromInt(math.NewInt(1000)), delegation.Shares)

	attackerAfter, found := k.GetPlayer(sdkCtx, attacker.Id)
	require.True(t, found)
	require.Empty(t, attackerAfter.GuildId, "the refused join must not have gone through")

	victimAfter, found := k.GetPlayer(sdkCtx, victim.Id)
	require.True(t, found)
	require.Empty(t, victimAfter.GuildId)
}

// TestGuildMembershipJoinAcceptsOwnedInfusionAddress is the positive control for
// the check above: the same flow, with the address belonging to the player who
// names it, still works. Without this the test above would pass just as well
// against a check that refuses everybody.
func TestGuildMembershipJoinAcceptsOwnedInfusionAddress(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	guild, found := k.GetGuild(sdkCtx, gs.Guild.Id)
	require.True(t, found)
	guild.JoinInfusionMinimum = 500
	k.SetGuild(sdkCtx, guild)

	joinerAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "joinownedjoiner")[:36])
	joiner := testAppendPlayer(k, ctx, types.Player{
		Creator:        joinerAcc.String(),
		PrimaryAddress: joinerAcc.String(),
	})

	sourceValAddr := sdk.ValAddress(fmt.Sprintf("%-36s", "joinownedsourceval")[:36])
	testAddValidator(k, sourceValAddr, math.NewInt(1000))
	sourceReactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:         sourceValAddr.String(),
		RawAddress:        sourceValAddr.Bytes(),
		DefaultCommission: math.LegacyZeroDec(),
	})

	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{
		DelegatorAddress: joinerAcc.String(),
		ValidatorAddress: sourceValAddr.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(1000)),
	}))

	testAppendInfusion(k, ctx, types.Infusion{
		DestinationId:   sourceReactor.Id,
		DestinationType: types.ObjectType_reactor,
		Address:         joinerAcc.String(),
		PlayerId:        joiner.Id,
		Fuel:            1000,
		Commission:      math.LegacyZeroDec(),
	})

	_, err := ms.GuildMembershipJoin(ctx, &types.MsgGuildMembershipJoin{
		Creator:    joiner.Creator,
		GuildId:    gs.Guild.Id,
		PlayerId:   joiner.Id,
		InfusionId: []string{sourceReactor.Id + "-" + joinerAcc.String()},
	})
	require.NoError(t, err)

	joined, found := k.GetPlayer(sdkCtx, joiner.Id)
	require.True(t, found)
	require.Equal(t, gs.Guild.Id, joined.GuildId)
}
