package keeper

import (
	"bytes"
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/prefix"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/BurntFinance/burnt/x/schedule/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type (
	Keeper struct {
		cdc                    codec.BinaryCodec
		storeKey               sdk.StoreKey
		memKey                 sdk.StoreKey
		paramstore             paramtypes.Subspace
		wasmViewKeeper         types.WasmViewKeeper
		wasmPermissionedKeeper types.WasmPermissionedKeeper
		feegrantKeeper         types.FeeGrantKeeper
		bankKeeper             types.BankKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,
	wasmViewKeeper types.WasmViewKeeper,
	wasmPermissionedKeeper types.WasmPermissionedKeeper,
	feegrantKeeper types.FeeGrantKeeper,
	bankKeeper types.BankKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:                    cdc,
		storeKey:               storeKey,
		memKey:                 memKey,
		paramstore:             ps,
		wasmViewKeeper:         wasmViewKeeper,
		wasmPermissionedKeeper: wasmPermissionedKeeper,
		feegrantKeeper:         feegrantKeeper,
		bankKeeper:             bankKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// Scheduled Calls

func (k Keeper) AddScheduledCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, callBody []byte, blockHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract, callBody)
	byNameKey := types.MakeScheduledCallByNameKey(signer, contract, callBody)

	value := &types.ScheduledCall{
		CallBody: callBody,
	}

	store.Set(byNameKey, sdk.Uint64ToBigEndian(blockHeight))
	store.Set(byHeightKey, k.cdc.MustMarshal(value))
}

func (k Keeper) RemoveScheduledCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, callBody []byte) {
	store := ctx.KVStore(k.storeKey)
	byNameKey := types.MakeScheduledCallByNameKey(signer, contract, callBody)

	blockHeight := sdk.BigEndianToUint64(store.Get(byNameKey))
	store.Delete(byNameKey)

	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract, callBody)
	store.Delete(byHeightKey)
}

func (k Keeper) RemoveScheduledCallWithBlockHeight(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, callBody []byte, blockHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	byNameKey := types.MakeScheduledCallByNameKey(signer, contract, callBody)
	store.Delete(byNameKey)

	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract, callBody)
	store.Delete(byHeightKey)
}

func (k Keeper) iterateScheduledCalls(ctx sdk.Context, cb func(height uint64, signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool)) {
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{types.ScheduledCallByBlockHeightKeyPrefix})
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		keyPair := bytes.NewBuffer(bytes.TrimPrefix(iter.Key(), []byte{types.ScheduledCallByBlockHeightKeyPrefix}))
		blockHeight := sdk.BigEndianToUint64(keyPair.Next(8))
		signer := sdk.AccAddress(keyPair.Next(20))
		contract := sdk.AccAddress(keyPair.Next(32))
		var call types.ScheduledCall
		k.cdc.MustUnmarshal(iter.Value(), &call)
		if cb(blockHeight, signer, contract, &call) {
			break
		}
	}
}

func (k Keeper) ConsumeScheduledCallsByHeight(ctx sdk.Context, blockHeight uint64, cb func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool)) {
	prefixKey := types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		keyPair := bytes.NewBuffer(bytes.TrimPrefix(iter.Key(), types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)))
		signer := sdk.AccAddress(keyPair.Next(20))
		contract := sdk.AccAddress(keyPair.Next(32))

		var call types.ScheduledCall
		k.cdc.MustUnmarshal(iter.Value(), &call)
		k.RemoveScheduledCallWithBlockHeight(ctx, signer, contract, call.CallBody, blockHeight)
		if cb(signer, contract, &call) {
			break
		}
	}
}
