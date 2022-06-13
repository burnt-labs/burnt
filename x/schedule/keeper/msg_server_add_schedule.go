package keeper

import (
	"context"

	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AddSchedule(goCtx context.Context, msg *types.MsgAddSchedule) (*types.MsgAddScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.BlockHeight <= uint64(ctx.BlockHeight()) {
		return nil, types.ErrInvalidScheduledBlockHeight
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, err
	}

	// todo: check that payer has grant from signer
	// todo: check that payer has minimum balance for call
	payer, err := sdk.AccAddressFromBech32(msg.Payer)
	if err != nil {
		return nil, err
	}

	contract, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, err
	}

	k.AddScheduledCall(ctx, signer, contract, msg.FunctionName, msg.BlockHeight, &payer)

	return &types.MsgAddScheduleResponse{}, nil
}
