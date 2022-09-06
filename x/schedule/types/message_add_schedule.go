package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddSchedule = "add_schedule"

var _ sdk.Msg = &MsgAddSchedule{}

func NewMsgAddSchedule(signer sdk.AccAddress, contract sdk.AccAddress, callBody []byte, blockHeight uint64) *MsgAddSchedule {
	return &MsgAddSchedule{
		Signer:      signer.String(),
		Contract:    contract.String(),
		CallBody:    callBody,
		BlockHeight: blockHeight,
	}
}

func (msg *MsgAddSchedule) Route() string {
	return RouterKey
}

func (msg *MsgAddSchedule) Type() string {
	return TypeMsgAddSchedule
}

func (msg *MsgAddSchedule) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgAddSchedule) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddSchedule) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}
	return nil
}
