package keeper

import (
	"context"

	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ScheduledCalls(c context.Context, req *types.QueryScheduledCallsRequest) (*types.QueryScheduledCallsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	var scheduledCalls []*types.QueryScheduledCall
	k.iterateScheduledCalls(ctx, func(height uint64, signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool) {
		scheduledCall := types.QueryScheduledCall{
			Contract:     contract.String(),
			FunctionName: call.FunctionName,
			Payer:        call.Payer,
			Height:       height,
			Signer:       signer,
		}
		scheduledCalls = append(scheduledCalls, &scheduledCall)
		return false
	})

	return &types.QueryScheduledCallsResponse{Calls: scheduledCalls}, nil
}
