package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadDeleteLeaderProposal = "squad_delete_leader_proposal"

var _ sdk.Msg = &MsgSquadDeleteLeaderProposal{}

func NewMsgSquadDeleteLeaderProposal(creator string, squadId uint64) *MsgSquadDeleteLeaderProposal {
	return &MsgSquadDeleteLeaderProposal{
		Creator:     creator,
		SquadId:     squadId,
	}
}

func (msg *MsgSquadDeleteLeaderProposal) Route() string {
	return RouterKey
}

func (msg *MsgSquadDeleteLeaderProposal) Type() string {
	return TypeMsgSquadDeleteLeaderProposal
}

func (msg *MsgSquadDeleteLeaderProposal) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadDeleteLeaderProposal) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadDeleteLeaderProposal) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
