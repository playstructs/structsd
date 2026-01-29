package types

import (
	"context"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

// ChannelKeeper defines the expected IBC channel keeper (IBC v10 - no capability params).
type ChannelKeeper interface {
	GetChannel(ctx context.Context, portID, channelID string) (channeltypes.Channel, bool)
	GetNextSequenceSend(ctx context.Context, portID, channelID string) (uint64, bool)
	SendPacket(
		ctx context.Context,
		sourcePort string,
		sourceChannel string,
		timeoutHeight clienttypes.Height,
		timeoutTimestamp uint64,
		data []byte,
	) (uint64, error)
	ChanCloseInit(ctx context.Context, portID, channelID string) error
}

// PortKeeper defines the expected IBC port keeper (IBC v10 - returns error, not capability).
type PortKeeper interface {
	BindPort(ctx context.Context, portID string) error
}
