package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateInfusionJoinMinimum = "guild_update_infusion_join_minimum"

var _ sdk.Msg = &MsgGuildUpdateInfusionJoinMinimum{}

func NewMsgGuildUpdateInfusionJoinMinimum(creator string, id uint64,  infusionJoinMinimum uint64) *MsgGuildUpdateInfusionJoinMinimum {
	return &MsgGuildUpdateInfusionJoinMinimum{
		Creator:  creator,
		Id: id,
		InfusionJoinMinimum: infusionJoinMinimum,
	}
}

func (msg *MsgGuildUpdateInfusionJoinMinimum) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateInfusionJoinMinimum) Type() string {
	return TypeMsgGuildUpdateInfusionJoinMinimum
}

func (msg *MsgGuildUpdateInfusionJoinMinimum) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateInfusionJoinMinimum) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateInfusionJoinMinimum) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
