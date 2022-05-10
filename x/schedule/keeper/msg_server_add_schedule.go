package keeper

import (
	"context"

	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AddSchedule(goCtx context.Context, msg *types.MsgAddSchedule) (*types.MsgAddScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	_ = ctx
	if msg.BlockHeight <= uint64(ctx.BlockHeight()) {
		return nil, types.ErrInvalidScheduledBlockHeight
	}

	return &types.MsgAddScheduleResponse{}, nil
}
