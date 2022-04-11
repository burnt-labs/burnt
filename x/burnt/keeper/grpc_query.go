package keeper

import (
	"github.com/BurntFinance/burnt/x/burnt/types"
)

var _ types.QueryServer = Keeper{}
