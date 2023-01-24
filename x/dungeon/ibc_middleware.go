package dungeon

import porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"

type IBCMiddleware struct {
	porttypes.IBCModule
}

func NewIBCMiddleware(app porttypes.IBCModule) porttypes.IBCModule {
	return IBCMiddleware{app}
}
