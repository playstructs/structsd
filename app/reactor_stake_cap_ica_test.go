package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/proto"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

// Seed the channel/account that a completed handshake would establish, then
// exercise the real ICA host receiver, signer checks, message router and SDK
// staking hooks. This covers host execution, not relaying or IBC proof checks.
func TestICAReactorStakeCapBatchRollbackAndSameContextRetry(t *testing.T) {
	for _, encoding := range []string{icatypes.EncodingProtobuf, icatypes.EncodingProto3JSON} {
		for _, allow := range []string{"*", sdk.MsgTypeURL(&stakingtypes.MsgDelegate{})} {
			t.Run(encoding+"/"+allow, func(t *testing.T) {
				bApp, ctx := setupJailGateAppWithValidators(t, 4)
				const connection, sourcePort, hostChannel = "connection-0", "icacontroller-cap-test", "channel-0"
				address := icatypes.GenerateAddress(ctx, connection, sourcePort)
				base := bApp.AccountKeeper.NewAccountWithAddress(ctx, address).(*authtypes.BaseAccount)
				bApp.AccountKeeper.SetAccount(ctx, icatypes.NewInterchainAccount(base, sourcePort))
				bApp.ICAHostKeeper.SetInterchainAccountAddress(ctx, connection, sourcePort, address.String())
				bApp.ICAHostKeeper.SetParams(ctx, icahosttypes.NewParams(true, []string{allow}))
				metadata := icatypes.NewMetadata(icatypes.Version, connection, connection, address.String(), encoding, icatypes.TxTypeSDKMultiMsg)
				channel := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED,
					channeltypes.NewCounterparty(sourcePort, "channel-1"), []string{connection}, string(icatypes.ModuleCdc.MustMarshalJSON(&metadata)))
				bApp.IBCKeeper.ChannelKeeper.SetChannel(ctx, icatypes.HostPortID, hostChannel, channel)
				validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
				require.NoError(t, err)
				validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
				require.NoError(t, err)
				denom, err := bApp.StakingKeeper.BondDenom(ctx)
				require.NoError(t, err)
				fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin(denom, 1000000)))
				activateTestStakeCap(t, bApp, ctx)
				execute := func(amounts ...int64) error {
					t.Helper()
					messages := make([]proto.Message, len(amounts))
					for i, amount := range amounts {
						messages[i] = stakingtypes.NewMsgDelegate(address.String(), validator.String(), sdk.NewInt64Coin(denom, amount))
					}
					wire, err := icatypes.SerializeCosmosTx(bApp.AppCodec(), messages, encoding)
					require.NoError(t, err)
					data := icatypes.InterchainAccountPacketData{Type: icatypes.EXECUTE_TX, Data: wire}
					_, err = bApp.ICAHostKeeper.OnRecvPacket(ctx, channeltypes.Packet{
						SourcePort: sourcePort, SourceChannel: "channel-1", DestinationPort: icatypes.HostPortID, DestinationChannel: hostChannel, Data: data.GetBytes(),
					})
					return err
				}
				before, nativeBefore := applicationAssetStateRows(bApp, ctx), bApp.StakingKeeper.ExportGenesis(ctx)
				events := len(ctx.EventManager().Events())
				require.ErrorIs(t, execute(100000, 200000), types.ErrReactorStakeCapExceeded)
				require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
				require.Equal(t, nativeBefore, bApp.StakingKeeper.ExportGenesis(ctx))
				require.Len(t, ctx.EventManager().Events(), events)
				// Reuse the same context: failed host execution must not leak its
				// consumed snapshot or first successful delegate into this packet.
				require.NoError(t, execute(100000, 180000))
				delegation, err := bApp.StakingKeeper.GetDelegation(ctx, address, validator)
				require.NoError(t, err)
				liveValidator, err := bApp.StakingKeeper.GetValidator(ctx, validator)
				require.NoError(t, err)
				require.Equal(t, "280000", liveValidator.TokensFromShares(delegation.Shares).TruncateInt().String())
				require.Equal(t, int64(720000), bApp.BankKeeper.GetBalance(ctx, address, denom).Amount.Int64())
				requireStakingAndDistributionHealthy(t, bApp, ctx)
			})
		}
	}
}
