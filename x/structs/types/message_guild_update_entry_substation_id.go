package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateEntrySubstationId = "guild_update_entry_substation_id"

var _ sdk.Msg = &MsgGuildUpdateEntrySubstationId{}

func NewMsgGuildUpdateEntrySubstationId(creator string, guildId string, substationId string) *MsgGuildUpdateEntrySubstationId {
	return &MsgGuildUpdateEntrySubstationId{
		Creator:  creator,
		GuildId: guildId,
		EntrySubstationId: substationId,
	}
}

func (msg *MsgGuildUpdateEntrySubstationId) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateEntrySubstationId) Type() string {
	return TypeMsgGuildUpdateEntrySubstationId
}

func (msg *MsgGuildUpdateEntrySubstationId) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateEntrySubstationId) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateEntrySubstationId) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
