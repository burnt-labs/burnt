package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/schedule module sentinel errors
var (
	ErrInvalidScheduledBlockHeight = sdkerrors.Register(ModuleName, 1100, "invalid scheduled block height")
	ErrUnmetMinimumBalance         = sdkerrors.Register(ModuleName, 1101, "unmet minimum balance")
	ErrUnauthorized                = sdkerrors.Register(ModuleName, 1102, "unauthorized")
	ErrTooFarInFuture              = sdkerrors.Register(ModuleName, 1103, "scheduled block height exceeds upper bound into future")
)
