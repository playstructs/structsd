package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadJoinRequest = "squad_join_request"

var _ sdk.Msg = &MsgSquadJoinRequest{}

func NewMsgSquadJoinRequest(creator string, squadId uint64, playerId uint64) *MsgSquadJoinRequest {
	return &MsgSquadJoinRequest{
		Creator:  creator,
		SquadId: squadId,
		PlayerId: playerId,
	}
}

func (msg *MsgSquadJoinRequest) Route() string {
	return RouterKey
}

func (msg *MsgSquadJoinRequest) Type() string {
	return TypeMsgSquadJoinRequest
}

func (msg *MsgSquadJoinRequest) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadJoinRequest) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadJoinRequest) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
