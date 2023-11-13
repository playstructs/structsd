package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadCreate = "squad_create"

var _ sdk.Msg = &MsgSquadCreate{}

func NewMsgSquadCreate(creator string, guildId uint64, leader uint64, squadJoinType uint64, entrySubstationId uint64) *MsgSquadCreate {
	return &MsgSquadCreate{
		Creator:  creator,
		GuildId: guildId,
		Leader: leader,
		SquadJoinType: squadJoinType,
		EntrySubstationId: entrySubstationId,
	}
}

func (msg *MsgSquadCreate) Route() string {
	return RouterKey
}

func (msg *MsgSquadCreate) Type() string {
	return TypeMsgSquadCreate
}

func (msg *MsgSquadCreate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadCreate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadCreate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
