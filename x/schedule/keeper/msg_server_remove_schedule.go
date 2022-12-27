package keeper

import (
	"context"
	"encoding/json"
	"github.com/burnt-labs/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) RemoveSchedule(goCtx context.Context, msg *types.MsgRemoveSchedule) (*types.MsgRemoveScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

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
	gasMinimum := k.GetParams(ctx).MinimumBalance
	balance := k.bankKeeper.GetBalance(ctx, contract, gasMinimum.Denom)

	k.RemoveScheduledCall(ctx, signer, contract)
	if err := ctx.EventManager().EmitTypedEvent(&types.RemoveScheduledCallEvent{
		BlockHeight: uint64(ctx.BlockHeight()),
		Signer:      signer.String(),
		Contract:    contract.String(),
		Balance:     &balance,
	}); err != nil {
		return nil, err
	}
	return &types.MsgRemoveScheduleResponse{}, nil
}
