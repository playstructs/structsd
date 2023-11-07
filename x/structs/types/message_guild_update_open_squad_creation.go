package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateOpenSquadCreation = "guild_update_open_squad_creation"

var _ sdk.Msg = &MsgGuildUpdateOpenSquadCreation{}

func NewMsgGuildUpdateOpenSquadCreation(creator string, id uint64, openSquadCreation bool) *MsgGuildUpdateOpenSquadCreation {
	return &MsgGuildUpdateOpenSquadCreation{
		Creator:  creator,
		Id: id,
		OpenSquadCreation: openSquadCreation,
	}
}

func (msg *MsgGuildUpdateOpenSquadCreation) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateOpenSquadCreation) Type() string {
	return TypeMsgGuildUpdateOpenSquadCreation
}

func (msg *MsgGuildUpdateOpenSquadCreation) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateOpenSquadCreation) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateOpenSquadCreation) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
