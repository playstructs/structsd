package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateJoinInfusionMinimum = "guild_update_join_infusion_minimum"

var _ sdk.Msg = &MsgGuildUpdateJoinInfusionMinimum{}

func NewMsgGuildUpdateJoinInfusionMinimum(creator string, id uint64,  joinInfusionMinimum uint64) *MsgGuildUpdateJoinInfusionMinimum {
	return &MsgGuildUpdateJoinInfusionMinimum{
		Creator:  creator,
		Id: id,
		JoinInfusionMinimum: joinInfusionMinimum,
	}
}

func (msg *MsgGuildUpdateJoinInfusionMinimum) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateJoinInfusionMinimum) Type() string {
	return TypeMsgGuildUpdateJoinInfusionMinimum
}

func (msg *MsgGuildUpdateJoinInfusionMinimum) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateJoinInfusionMinimum) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateJoinInfusionMinimum) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
