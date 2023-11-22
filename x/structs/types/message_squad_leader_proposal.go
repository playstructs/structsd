package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadLeaderProposal = "squad_leader_proposal"

var _ sdk.Msg = &MsgSquadLeaderProposal{}

func NewMsgSquadLeaderProposal(creator string, squadId uint64, leader uint64) *MsgSquadLeaderProposal {
	return &MsgSquadLeaderProposal{
		Creator:  creator,
		SquadId: squadId,
		Leader: leader,
	}
}

func (msg *MsgSquadLeaderProposal) Route() string {
	return RouterKey
}

func (msg *MsgSquadLeaderProposal) Type() string {
	return TypeMsgSquadLeaderProposal
}

func (msg *MsgSquadLeaderProposal) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadLeaderProposal) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadLeaderProposal) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
