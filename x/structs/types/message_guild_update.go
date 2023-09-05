package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdate = "guild_update"

var _ sdk.Msg = &MsgGuildUpdate{}

func NewMsgGuildUpdate(creator string, id uint64, endpoint string, substationId uint64, owner uint64, guildJoinType uint64, infusionJoinMinimum uint64) *MsgGuildUpdate {
	return &MsgGuildUpdate{
		Creator:  creator,
		Id: id,
		Endpoint: endpoint,
		EntrySubstationId: substationId,
		Owner: owner, 
		GuildJoinType: guildJoinType,
		InfusionJoinMinimum: infusionJoinMinimum,
	}
}

func (msg *MsgGuildUpdate) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdate) Type() string {
	return TypeMsgGuildUpdate
}

func (msg *MsgGuildUpdate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
