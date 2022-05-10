package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/schedule module sentinel errors
var (
	ErrInvalidScheduledBlockHeight = sdkerrors.Register(ModuleName, 1100, "invalid scheduled block height")
)
