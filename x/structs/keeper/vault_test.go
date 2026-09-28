package keeper_test

import (
	"sort"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"structs/x/structs/types"
)

func TestVaultQueryIsReadOnlyAndReportsProvisioning(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	_, err := f.k.Vault(f.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	for _, id := range []string{"0-99999999", "9-1", "1-01"} {
		_, err := f.k.Vault(f.ctx, &types.QueryVaultRequest{OwnerId: id})
		require.Equal(t, codes.NotFound, status.Code(err))
	}
	request := &types.QueryVaultRequest{OwnerId: f.guild.Id}
	before, err := f.k.Vault(f.ctx, request)
	require.NoError(t, err)
	require.False(t, before.Provisioned)
	address, err := types.VaultAddress(f.guild.Id)
	require.NoError(t, err)
	require.Equal(t, address.String(), before.Address)
	require.Nil(t, f.k.AccountKeeper().GetAccount(f.ctx, address))
	_, err = f.k.NewCurrentContext(f.ctx).EnsureVaults(f.guild.Id)
	require.NoError(t, err)
	after, err := f.k.Vault(f.ctx, request)
	require.NoError(t, err)
	require.True(t, after.Provisioned)
	require.Equal(t, before.OwnerId, after.OwnerId)
	require.Equal(t, before.Address, after.Address)
}

func TestVaultLookupDoesNotProvisionOrCreateSigningIdentity(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	cc := f.k.NewCurrentContext(f.ctx)
	vault, err := cc.GetVault(f.guild.Id)
	require.NoError(t, err)
	require.Same(t, cc.GetGuild(f.guild.Id), vault.Owner())
	same, err := cc.GetVault(f.guild.Id)
	require.NoError(t, err)
	require.Same(t, vault, same)
	cc.CommitAll()
	require.Nil(t, f.k.AccountKeeper().GetAccount(f.ctx, vault.Address()))
	_, found := f.k.GetVaultOwner(f.ctx, vault.Address())
	require.False(t, found)
	require.Zero(t, f.k.GetPlayerIndexFromAddress(f.ctx, vault.Address().String()))
	require.Empty(t, cc.SignerAddress())
}

func TestVaultProvisioningIsSortedIdempotentAndIndexed(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	cc := f.k.NewCurrentContext(f.ctx)
	ids := []string{f.guild.Id, f.seller.Id, f.buyer.Id}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	addresses, err := cc.EnsureVaults(ids...)
	require.NoError(t, err)
	byID := make(map[string]uint64)
	for i, address := range addresses {
		id, found := f.k.GetVaultOwner(f.ctx, address)
		require.True(t, found)
		require.Equal(t, ids[i], id)
		account := f.k.AccountKeeper().GetAccount(f.ctx, address)
		require.NotNil(t, account)
		require.Nil(t, account.GetPubKey())
		byID[id] = account.GetAccountNumber()
		owner, err := cc.GetAssetOwnerByAddress(address)
		require.NoError(t, err)
		require.Equal(t, id, owner.ID())
		require.Zero(t, f.k.GetPlayerIndexFromAddress(f.ctx, address.String()))
	}
	sort.Strings(ids)
	for i := 1; i < len(ids); i++ {
		require.Less(t, byID[ids[i-1]], byID[ids[i]])
	}
	_, err = cc.EnsureVaults(ids...)
	require.NoError(t, err)
	cc.CommitAll()
	for _, id := range ids {
		address, err := types.VaultAddress(id)
		require.NoError(t, err)
		require.Equal(t, byID[id], f.k.AccountKeeper().GetAccount(f.ctx, address).GetAccountNumber())
	}
	eventOwners := []string{}
	for _, event := range f.ctx.EventManager().ABCIEvents() {
		if event.Type == "structs.structs.EventAddressAssociation" { t.Fatal("vault provisioning must not advertise a signing-key association") }
		if event.Type != "structs.structs.EventVaultAddress" { continue }
		parsed, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		detail := parsed.(*types.EventVaultAddress).EventVaultAddressDetail
		eventOwners = append(eventOwners, detail.OwnerId)
		address, err := types.VaultAddress(detail.OwnerId)
		require.NoError(t, err)
		require.Equal(t, address.String(), detail.VaultAddress)
		require.Equal(t, byID[detail.OwnerId], detail.AccountNumber)
		require.True(t, detail.AccountCreated)
	}
	require.Equal(t, ids, eventOwners, "one association event per vault in deterministic order, with no duplicates on rerun")
}

func TestVaultProvisioningPreflightsEveryOwner(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	cc := f.k.NewCurrentContext(f.ctx)
	_, err := cc.EnsureVaults(f.buyer.Id, "0-99999999")
	require.Error(t, err)
	cc.CommitAll()
	address, err := types.VaultAddress(f.buyer.Id)
	require.NoError(t, err)
	require.Nil(t, f.k.AccountKeeper().GetAccount(f.ctx, address))
	_, found := f.k.GetVaultOwner(f.ctx, address)
	require.False(t, found)
}

func TestVaultHolderEligibilityRequiresProvisionedProtocolAddress(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	cc := f.k.NewCurrentContext(f.ctx)
	address, err := types.VaultAddress(f.guild.Id)
	require.NoError(t, err)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uguild."+f.guild.Id, 1))
	_, err = f.k.GuildBankDenomSendRestriction(f.ctx, nil, address, coins)
	require.Error(t, err)
	_, err = cc.EnsureVaults(f.guild.Id, f.buyer.Id)
	require.NoError(t, err)
	for _, id := range []string{f.guild.Id, f.buyer.Id} {
		address, err := types.VaultAddress(id)
		require.NoError(t, err)
		got, err := f.k.GuildBankDenomSendRestriction(f.ctx, nil, address, coins)
		require.NoError(t, err)
		require.Equal(t, address, got)
	}
	_, err = f.k.GuildBankDenomSendRestriction(f.ctx, nil, sdk.AccAddress("unregistered-holder!"), coins)
	require.Error(t, err)
}

func TestGuildVaultAuthorityFollowsOwnerWithoutBecomingSigner(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	cc := f.k.NewCurrentContext(f.ctx)
	actor, err := cc.GetSigningPlayer(f.buyer.PrimaryAddress)
	require.NoError(t, err)
	vault, err := cc.GetVault(f.guild.Id)
	require.NoError(t, err)
	require.NoError(t, vault.CanUseBy(actor, types.PermTokenTransfer))
	require.Error(t, vault.CanUseBy(cc.GetPlayer(f.member.Id), types.PermTokenTransfer))
	require.Equal(t, f.buyer.PrimaryAddress, cc.SignerAddress())
	_, err = cc.EnsureVaults(f.guild.Id)
	require.NoError(t, err)
	_, err = f.k.NewCurrentContext(f.ctx).GetSigningPlayer(vault.Address().String())
	require.Error(t, err)
}
