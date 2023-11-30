package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadDeleteJoinRequest = "squad_delete_join_request"

var _ sdk.Msg = &MsgSquadDeleteJoinRequest{}

func NewMsgSquadDeleteJoinRequest(creator string, squadId uint64, playerId uint64) *MsgSquadDeleteJoinRequest {
	return &MsgSquadDeleteJoinRequest{
		Creator:  creator,
		SquadId: squadId,
		PlayerId: playerId,
	}
}

func (msg *MsgSquadDeleteJoinRequest) Route() string {
	return RouterKey
}

func (msg *MsgSquadDeleteJoinRequest) Type() string {
	return TypeMsgSquadDeleteJoinRequest
}

func (msg *MsgSquadDeleteJoinRequest) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadDeleteJoinRequest) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadDeleteJoinRequest) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
