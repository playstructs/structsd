package app_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	icagenesistypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestICAHostDisabledInApplicationAndCLIGenesis(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	modules := app.RegisterIBC(codectypes.NewInterfaceRegistry())
	cli := module.CoreAppModuleBasicAdaptor(icatypes.ModuleName, modules[icatypes.ModuleName])
	cliGenesis := module.BasicManager{icatypes.ModuleName: cli}.DefaultGenesis(bApp.AppCodec())
	for _, raw := range [][]byte{bApp.DefaultGenesis()[icatypes.ModuleName], cliGenesis[icatypes.ModuleName]} {
		var state icagenesistypes.GenesisState
		bApp.AppCodec().MustUnmarshalJSON(raw, &state)
		require.False(t, state.HostGenesisState.Params.HostEnabled)
		require.Empty(t, state.HostGenesisState.Params.AllowMessages)
	}
	params := bApp.ICAHostKeeper.GetParams(ctx)
	require.False(t, params.HostEnabled, "InitChain must install the application defaults")
	require.Empty(t, params.AllowMessages)
}

func TestICAHostPolicyRejectsHandshakeAndDispatch(t *testing.T) {
	for _, migrate := range []bool{false, true} {
		t.Run(map[bool]string{false: "genesis", true: "upgrade"}[migrate], func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			transferBefore := bApp.TransferKeeper.GetParams(ctx)
			controllerBefore := bApp.ICAControllerKeeper.GetParams(ctx)
			const connection, sourcePort, hostChannel = "connection-0", "icacontroller-policy", "channel-0"
			address := icatypes.GenerateAddress(ctx, connection, sourcePort)
			base := bApp.AccountKeeper.NewAccountWithAddress(ctx, address).(*authtypes.BaseAccount)
			bApp.AccountKeeper.SetAccount(ctx, icatypes.NewInterchainAccount(base, sourcePort))
			bApp.ICAHostKeeper.SetInterchainAccountAddress(ctx, connection, sourcePort, address.String())
			metadata := icatypes.NewMetadata(icatypes.Version, connection, connection, address.String(), icatypes.EncodingProtobuf, icatypes.TxTypeSDKMultiMsg)
			channel := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED,
				channeltypes.NewCounterparty(sourcePort, "channel-1"), []string{connection}, string(icatypes.ModuleCdc.MustMarshalJSON(&metadata)))
			bApp.IBCKeeper.ChannelKeeper.SetChannel(ctx, icatypes.HostPortID, hostChannel, channel)
			if migrate {
				bApp.ICAHostKeeper.SetParams(ctx, icahosttypes.NewParams(true, []string{"*"}))
				v22.MigrateICAHostPolicy(ctx, delegationUpgradeKeepers(bApp))
			}
			require.Equal(t, transferBefore, bApp.TransferKeeper.GetParams(ctx))
			require.True(t, transferBefore.SendEnabled)
			require.True(t, transferBefore.ReceiveEnabled)
			require.Equal(t, controllerBefore, bApp.ICAControllerKeeper.GetParams(ctx))
			require.True(t, bApp.IBCKeeper.PortKeeper.Router.HasRoute(transfertypes.ModuleName))
			route, found := bApp.IBCKeeper.PortKeeper.Router.Route(icahosttypes.SubModuleName)
			require.True(t, found)
			_, err := route.OnChanOpenTry(ctx, channeltypes.ORDERED, []string{connection}, icatypes.HostPortID, "channel-2", channel.Counterparty, channel.Version)
			require.ErrorIs(t, err, icahosttypes.ErrHostSubModuleDisabled)
			require.ErrorIs(t, route.OnChanOpenConfirm(ctx, icatypes.HostPortID, hostChannel), icahosttypes.ErrHostSubModuleDisabled)
			gameplay := &types.MsgStructAttack{Creator: address.String(), OperatingStructId: "5-1", TargetStructId: []string{"5-2"}, WeaponSystem: "primaryWeapon"}
			wrapped := authz.NewMsgExec(address, []sdk.Msg{gameplay})
			messages := []proto.Message{gameplay, &wrapped, &banktypes.MsgSend{FromAddress: address.String(), ToAddress: address.String(), Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1))}}
			before := applicationAssetStateRows(bApp, ctx)
			for _, msg := range messages {
				wire, err := icatypes.SerializeCosmosTx(bApp.AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				require.NoError(t, err)
				data := icatypes.InterchainAccountPacketData{Type: icatypes.EXECUTE_TX, Data: wire}
				packet := channeltypes.Packet{SourcePort: sourcePort, SourceChannel: "channel-1", DestinationPort: icatypes.HostPortID, DestinationChannel: hostChannel, Data: data.GetBytes()}
				ack := route.OnRecvPacket(ctx, channel.Version, packet, nil)
				require.Equal(t, channeltypes.NewErrorAcknowledgement(icahosttypes.ErrHostSubModuleDisabled).Acknowledgement(), ack.Acknowledgement())
				require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
				// Even an accidental host-only re-enable must not restore the
				// wildcard. Exercise the real decoder and allow-list gate too.
				params := bApp.ICAHostKeeper.GetParams(ctx)
				params.HostEnabled = true
				bApp.ICAHostKeeper.SetParams(ctx, params)
				_, err = bApp.ICAHostKeeper.OnRecvPacket(ctx, packet)
				require.ErrorContains(t, err, "message type not allowed")
				params.HostEnabled = false
				bApp.ICAHostKeeper.SetParams(ctx, params)
			}
			stored, found := bApp.IBCKeeper.ChannelKeeper.GetChannel(ctx, icatypes.HostPortID, hostChannel)
			require.True(t, found)
			require.Equal(t, channel, stored, "policy changes preserve existing channel records")
			storedAddress, found := bApp.ICAHostKeeper.GetInterchainAccountAddress(ctx, connection, sourcePort)
			require.True(t, found)
			require.Equal(t, address.String(), storedAddress)
		})
	}
}
