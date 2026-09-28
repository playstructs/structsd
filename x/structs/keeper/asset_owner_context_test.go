package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestAssetOwnerResolutionReusesCachesWithoutChangingSigner(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	cc := f.k.NewCurrentContext(f.ctx)
	actor, err := cc.GetSigningPlayer(f.buyer.PrimaryAddress)
	require.NoError(t, err)

	player, err := cc.GetExistingAssetOwner(f.seller.Id)
	require.NoError(t, err)
	require.Same(t, cc.GetPlayer(f.seller.Id), player)
	guild, err := cc.GetExistingAssetOwner(f.guild.Id)
	require.NoError(t, err)
	require.Same(t, cc.GetGuild(f.guild.Id), guild)
	require.Equal(t, f.buyer.PrimaryAddress, cc.SignerAddress())
	require.NoError(t, cc.PermissionCheck(guild, actor, types.PermAdmin),
		"the owner need not be a guild member")

	// Ownership never lets a secondary key exceed its own permission bits.
	cc.ClearPermissions(keeperlib.GetAddressPermissionIDBytes(f.buyer.PrimaryAddress))
	require.Error(t, cc.PermissionCheck(guild, actor, types.PermAdmin))
}

func TestAssetOwnerResolutionRejectsPhantomsAndOtherObjectTypes(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	cc := f.k.NewCurrentContext(f.ctx)
	for _, id := range []string{"1-99999999", "0-99999999", "3-1", "9-1", "1-01", ""} {
		owner, err := cc.GetExistingAssetOwner(id)
		require.Error(t, err, id)
		require.Nil(t, owner, "failed resolution must not return a typed nil or phantom")
	}
	cc.CommitAll()
	_, found := f.k.GetPlayer(f.ctx, "1-99999999")
	require.False(t, found)
	_, found = f.k.GetGuild(f.ctx, "0-99999999")
	require.False(t, found)
}
