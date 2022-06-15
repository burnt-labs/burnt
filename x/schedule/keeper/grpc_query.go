package keeper

import (
	"github.com/BurntFinance/burnt/x/schedule/types"
)

var _ types.QueryServer = Keeper{}
