package keeper

import (
	"github.com/burnt-labs/burnt/x/schedule/types"
)

var _ types.QueryServer = Keeper{}
