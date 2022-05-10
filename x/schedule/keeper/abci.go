package keeper

import (
	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) EndBlocker(ctx sdk.Context) {

	k.ConsumeScheduledCallsByHeight(ctx, uint64(ctx.BlockHeight()), func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool) {
		// do something with the calls
		return false
	})
}