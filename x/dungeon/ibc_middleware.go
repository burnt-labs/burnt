package dungeon

import (
	"github.com/burnt-labs/burnt/x/dungeon/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v4/modules/core/exported"
)

type IBCMiddleware struct {
	porttypes.IBCModule
	keeper Keeper
}

type Keeper struct {
	porttypes.ICS4Wrapper
	stakingKeeper types.StakingKeeper
}

// NewKeeper creates a new dungeon Keeper instance.
func NewKeeper(wrapper porttypes.ICS4Wrapper, stakingKeeper *stakingkeeper.Keeper) Keeper {
	return Keeper{
		ICS4Wrapper:   wrapper,
		stakingKeeper: stakingKeeper,
	}
}

func NewIBCMiddleware(app porttypes.IBCModule, wrapper porttypes.ICS4Wrapper, stakingKeeper *stakingkeeper.Keeper) porttypes.IBCModule {
	return IBCMiddleware{app, NewKeeper(wrapper, stakingKeeper)}
}

func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
) error {
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		// If this happens either a) a user has crafted an invalid packet, b) a
		// software developer has connected the middleware to a stack that does
		// not have a transfer module, or c) the transfer module has been modified
		// to accept other Packets. The best thing we can do here is pass the packet
		// on down the stack.
		return im.keeper.SendPacket(ctx, chanCap, packet)
	}

	// checks to make sure that the sending chain is our chain
	if transfertypes.SenderChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.Denom) {

		// if the denomination of the token being sent is our bond token, block it
		if data.Denom == im.keeper.stakingKeeper.BondDenom(ctx) {
			return types.ErrTokenTransferBlocked
		}

		// otherwise, send the token
		return im.keeper.SendPacket(ctx, chanCap, packet)
	}

	return im.keeper.SendPacket(ctx, chanCap, packet)
}
