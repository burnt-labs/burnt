package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgRemoveSchedule = "remove_schedule"

var _ sdk.Msg = &MsgRemoveSchedule{}

func NewMsgRemoveSchedule(signer sdk.AccAddress, contract sdk.AccAddress) *MsgRemoveSchedule {
	return &MsgRemoveSchedule{
		Signer:   signer.String(),
		Contract: contract.String(),
	}
}

func (msg *MsgRemoveSchedule) Route() string {
	return RouterKey
}

func (msg *MsgRemoveSchedule) Type() string {
	return TypeMsgRemoveSchedule
}

func (msg *MsgRemoveSchedule) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgRemoveSchedule) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRemoveSchedule) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Contract); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid contract address (%s)", err)
	}

	return nil
}
