package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildApproveRegister = "guild_approve_register"

var _ sdk.Msg = &MsgGuildApproveRegister{}

func NewMsgGuildApproveRegister(creator string, approve bool, guildId uint64, playerId string) *MsgGuildApproveRegister {
	return &MsgGuildApproveRegister{
		Creator:     creator,
		Approve:     approve,
		GuildId:     guildId,
        PlayerId:    playerId,
	}
}

func (msg *MsgGuildApproveRegister) Route() string {
	return RouterKey
}

func (msg *MsgGuildApproveRegister) Type() string {
	return TypeMsgGuildApproveRegister
}

func (msg *MsgGuildApproveRegister) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildApproveRegister) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildApproveRegister) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
