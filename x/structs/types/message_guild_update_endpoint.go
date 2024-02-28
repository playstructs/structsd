package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateEndpoint = "guild_update_endpoint"

var _ sdk.Msg = &MsgGuildUpdateEndpoint{}

func NewMsgGuildUpdateEndpoint(creator string, guildId string, endpoint string) *MsgGuildUpdateEndpoint {
	return &MsgGuildUpdateEndpoint{
		Creator:  creator,
		GuildId: guildId,
		Endpoint: endpoint,
	}
}

func (msg *MsgGuildUpdateEndpoint) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateEndpoint) Type() string {
	return TypeMsgGuildUpdateEndpoint
}

func (msg *MsgGuildUpdateEndpoint) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateEndpoint) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateEndpoint) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
