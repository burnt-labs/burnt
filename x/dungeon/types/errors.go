package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

const ModuleName = "dungeon"

var (
	ErrTokenTransferBlocked = sdkerrors.Register(ModuleName, 1, "IBC transfers of this token are currently blocked")
)
