package keeper

import (
	"context"
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
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{

		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// Scheduled Calls

func (k Keeper) AddScheduledCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, functionName string, blockHeight uint64, payer *sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract, functionName)
	byNameKey := types.MakeScheduledCallByNameKey(signer, contract, functionName)

	value := &types.ScheduledCall{
		FunctionName: functionName,
		Payer:        signer.Bytes(),
	}

	if payer != nil {
		value.Payer = payer.Bytes()
	}

	store.Set(byNameKey, sdk.Uint64ToBigEndian(blockHeight))
	store.Set(byHeightKey, k.cdc.MustMarshal(value))
}

func (k Keeper) RemoveScheduledCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, functionName string) {
	store := ctx.KVStore(k.storeKey)
	byNameKey := types.MakeScheduledCallByNameKey(signer, contract, functionName)

	blockHeight := sdk.BigEndianToUint64(store.Get(byNameKey))
	store.Delete(byNameKey)

	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract, functionName)
	store.Delete(byHeightKey)
}

func (k Keeper) IterateScheduledCallsByHeight(ctx sdk.Context, blockHeight uint64, cb func(key []byte, call *types.ScheduledCall) (stop bool)) {
	prefixKey := types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var call types.ScheduledCall
		k.cdc.MustUnmarshal(iter.Value(), &call)
		if cb(iter.Key(), &call) {
			break
		}
	}
}

type ScheduledCallPair struct {
	Key  []byte
	Call *types.ScheduledCall
}

func (k Keeper) ScheduledCallsByHeight(ctx sdk.Context, blockHeight uint64) (<-chan ScheduledCallPair, func()) {
	prefixKey := types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixKey)
	iter := prefixStore.Iterator(nil, nil)
	c, cancel := context.WithCancel(ctx.Context())
	pairs := make(chan ScheduledCallPair, 1)
	go func() {
		defer iter.Close()
		defer close(pairs)
		for ; iter.Valid(); iter.Next() {
			select {
			case <-c.Done():
				break
			default:
				var call types.ScheduledCall
				k.cdc.MustUnmarshal(iter.Value(), &call)
				pairs <- ScheduledCallPair{
					Key:  iter.Key(),
					Call: &call,
				}
			}
		}
	}()
	return pairs, cancel
}
