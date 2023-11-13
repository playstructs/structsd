package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadApproveLeaderProposal = "squad_approve_leader_proposal"

var _ sdk.Msg = &MsgSquadApproveLeaderProposal{}

func NewMsgSquadApproveLeaderProposal(creator string, squadId uint64, playerId uint64, approve bool) *MsgSquadApproveLeaderProposal {
	return &MsgSquadApproveLeaderProposal{
		Creator:     creator,
		Approve:     approve,
		SquadId:     squadId,
        PlayerId:    playerId,
	}
}

func (msg *MsgSquadApproveLeaderProposal) Route() string {
	return RouterKey
}

func (msg *MsgSquadApproveLeaderProposal) Type() string {
	return TypeMsgSquadApproveLeaderProposal
}

func (msg *MsgSquadApproveLeaderProposal) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadApproveLeaderProposal) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadApproveLeaderProposal) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
