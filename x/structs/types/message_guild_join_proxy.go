package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildJoinProxy = "guild_join_proxy"

var _ sdk.Msg = &MsgGuildJoinProxy{}

func NewMsgGuildJoinProxy(creator string, address string, substationId string) *MsgGuildJoinProxy {
	return &MsgGuildJoinProxy{
		Creator:      creator,
		Address:      address,
		SubstationId: substationId,
	}
}

func (msg *MsgGuildJoinProxy) Route() string {
	return RouterKey
}

func (msg *MsgGuildJoinProxy) Type() string {
	return TypeMsgGuildJoinProxy
}

func (msg *MsgGuildJoinProxy) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildJoinProxy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildJoinProxy) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
