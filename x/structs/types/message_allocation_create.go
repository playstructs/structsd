package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strings"
	"strconv"
)

const TypeMsgAllocationCreate = "allocation_create"

var _ sdk.Msg = &MsgAllocationCreate{}

func NewMsgAllocationCreate(creator string, controller string, sourceObjectId string, power uint64, allocationType AllocationType) *MsgAllocationCreate {
	return &MsgAllocationCreate{
		Creator:        creator,
		Controller:     controller,
		SourceObjectId: sourceObjectId,
		Power:          power,
		AllocationType: allocationType,
	}
}

func (msg *MsgAllocationCreate) Route() string {
	return RouterKey
}

func (msg *MsgAllocationCreate) Type() string {
	return TypeMsgAllocationCreate
}

func (msg *MsgAllocationCreate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAllocationCreate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAllocationCreate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	sourceSplit := strings.Split(msg.SourceObjectId, "-")
	number, _ := strconv.ParseUint(sourceSplit[0], 10, 64)
    sourceType := ObjectType(number)
    if !IsValidAllocationConnectionType(sourceType) {
        return sdkerrors.Wrapf(ErrAllocationSourceType, "source type (%s) not valid for allocation", sourceType.String())
    }

	return nil
}
