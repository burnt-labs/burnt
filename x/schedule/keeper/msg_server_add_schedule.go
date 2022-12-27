package keeper

import (
	"context"
	"encoding/json"
	"github.com/burnt-labs/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type isOwnerResponse struct {
	IsOwner bool `json:"is_owner"`
}

func (k msgServer) AddSchedule(goCtx context.Context, msg *types.MsgAddSchedule) (*types.MsgAddScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.BlockHeight <= uint64(ctx.BlockHeight()) {
		return nil, types.ErrInvalidScheduledBlockHeight
	}

	if msg.BlockHeight > (uint64(ctx.BlockHeight()) + k.GetParams(ctx).UpperBound) {
		return nil, types.ErrTooFarInFuture
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, err
	}

	contract, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, err
	}

	ownerQueryMsg, err := json.Marshal(map[string]interface{}{
		"is_owner": map[string]interface{}{
			"address": signer,
		},
	})
	ownerQueryRes, err := k.wasmViewKeeper.QuerySmart(ctx, contract, ownerQueryMsg)
	if err != nil {
		return nil, err
	}

	var isOwner isOwnerResponse
	err = json.Unmarshal(ownerQueryRes, &isOwner)
	if err != nil {
		return nil, err
	}
	if !isOwner.IsOwner {
		return nil, types.ErrUnauthorized
	}

	// todo: anti-spam protection. how do we keep this from getting blown up for free?
	// probably we just charge gas for this

	gasMinimum := k.GetParams(ctx).MinimumBalance
	balance := k.bankKeeper.GetBalance(ctx, contract, gasMinimum.Denom)

	if balance.Amount.LT(gasMinimum.Amount) {
		// the contract doesn't have the funds
		return nil, types.ErrUnmetMinimumBalance
	}

	if existingScheduledBlockHeight := k.BlockHeightForSignerContract(ctx, signer, contract); existingScheduledBlockHeight != 0 {
		k.ReScheduleCall(ctx, signer, contract, msg.CallBody, existingScheduledBlockHeight, msg.BlockHeight)
	} else {
		k.AddScheduledCall(ctx, signer, contract, msg.CallBody, msg.BlockHeight)
	}
	if err := ctx.EventManager().EmitTypedEvent(&types.AddScheduledCallEvent{
		BlockHeight:     uint64(ctx.BlockHeight()),
		ScheduledHeight: msg.BlockHeight,
		Signer:          signer.String(),
		Contract:        contract.String(),
		Balance:         &balance,
		CallBody:        msg.CallBody,
	}); err != nil {
		return nil, err
	}
	return &types.MsgAddScheduleResponse{}, nil
}
