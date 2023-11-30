package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadApproveJoinRequest = "squad_approve_join_request"

var _ sdk.Msg = &MsgSquadApproveJoinRequest{}

func NewMsgSquadApproveJoinRequest(creator string, squadId uint64, playerId uint64, approve bool) *MsgSquadApproveJoinRequest {
	return &MsgSquadApproveJoinRequest{
		Creator:     creator,
		Approve:     approve,
		SquadId:     squadId,
        PlayerId:    playerId,
	}
}

func (msg *MsgSquadApproveJoinRequest) Route() string {
	return RouterKey
}

func (msg *MsgSquadApproveJoinRequest) Type() string {
	return TypeMsgSquadApproveJoinRequest
}

func (msg *MsgSquadApproveJoinRequest) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadApproveJoinRequest) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadApproveJoinRequest) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
