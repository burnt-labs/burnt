package ibc_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const StartCount = 1337

func TestScheduledCall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	ctx := context.Background()

	var numFullNodes = 1
	var numValidators = 3

	// pulling image from env to foster local dev
	imageTag := os.Getenv("BURNT_IMAGE")
	imageTagComponents := strings.Split(imageTag, ":")

	cf := ibctest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*ibctest.ChainSpec{
		{
			Name:    imageTagComponents[0],
			Version: imageTagComponents[1],
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: imageTagComponents[0],
						Version:    imageTagComponents[1],
						UidGid:     "1025:1025",
					},
				},
				GasPrices:      "0.0uburnt",
				GasAdjustment:  3,
				Type:           "cosmos",
				ChainID:        "burnt-1",
				Bin:            "burntd",
				Bech32Prefix:   "burnt",
				Denom:          "uburnt",
				TrustingPeriod: "336h",
				ModifyGenesis:  modifyGenesisShortProposals(votingPeriod, maxDepositPeriod),
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	burnt := chains[0].(*cosmos.CosmosChain)
	client, network := ibctest.DockerSetup(t)

	// Log location
	f, err := ibctest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	// Reporter/logs
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Prep Interchain
	ic := ibctest.NewInterchain().AddChain(burnt)

	// Build Interchain
	require.NoError(t, ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: ibctest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false},
	),
	)

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, burnt)
	burntUser := users[0]
	t.Logf("created burnt user %s", burntUser.FormattedAddress())

	burntUserBalInitial, err := burnt.GetBalance(ctx, burntUser.FormattedAddress(), burnt.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, burntUserBalInitial)

	// test begins - hold onto your whatever

	// upload contracts
	tickerCodeID, err := burnt.StoreContract(ctx, burntUser.KeyName(), "contracts/compiled/ticker.wasm")
	require.NoError(t, err)
	require.NotZero(t, tickerCodeID)
	proxyCodeID, err := burnt.StoreContract(ctx, burntUser.KeyName(), "contracts/compiled/proxy.wasm")
	require.NoError(t, err)
	require.NotZero(t, proxyCodeID)

	// instantiate contracts
	tickerInstantiateMsg, err := json.Marshal(map[string]interface{}{
		"count": StartCount,
	})
	require.NoError(t, err)
	adminAddr := burntUser.FormattedAddress()
	tickerContractAddr, err := burnt.InstantiateContract(ctx, burntUser.KeyName(), tickerCodeID, string(tickerInstantiateMsg), &adminAddr)
	require.NoError(t, err)

	proxyInstantiateMsg, err := json.Marshal(map[string]interface{}{})
	require.NoError(t, err)
	proxyContractAddr, err := burnt.InstantiateContract(ctx, burntUser.KeyName(), proxyCodeID, string(proxyInstantiateMsg), &adminAddr)
	require.NoError(t, err)

	// query initial contract state
	count, err := queryTickerCount(ctx, burnt, tickerContractAddr)
	require.NoError(t, err)
	require.Equal(t, StartCount, count)

	isOwner, err := queryIsProxyOwner(ctx, burnt, proxyContractAddr, burntUser.Address())
	require.NoError(t, err)
	require.True(t, isOwner)

	// start prodding the contracts
	incrementMsg, err := json.Marshal(map[string]interface{}{
		"increment": map[string]interface{}{},
	})
	require.NoError(t, err)
	require.NoError(t, burnt.ExecuteContract(ctx, burntUser.KeyName(), tickerContractAddr, string(incrementMsg)))
}

type TickerCountQueryResponse struct {
	Data CountResponse `json:"data"`
}
type CountResponse struct {
	Count int32 `json:"count"`
}

func queryTickerCount(ctx context.Context, chain *cosmos.CosmosChain, addr string) (int, error) {
	queryData := map[string]interface{}{
		"get_count": map[string]interface{}{},
	}
	var res TickerCountQueryResponse
	if err := chain.QueryContract(ctx, addr, queryData, &res); err != nil {
		return 0, err
	}

	return int(res.Data.Count), nil
}

type ProxyIsOwnerQueryResponse struct {
	Data IsOwnerResponse `json:"data"`
}
type IsOwnerResponse struct {
	IsOwner bool `json:"is_owner"`
}

func queryIsProxyOwner(ctx context.Context, chain *cosmos.CosmosChain, addr string, owner sdktypes.AccAddress) (bool, error) {
	queryData := map[string]interface{}{
		"is_owner": map[string]interface{}{
			"address": owner,
		},
	}
	var res ProxyIsOwnerQueryResponse
	if err := chain.QueryContract(ctx, addr, queryData, &res); err != nil {
		return false, err
	}
	return res.Data.IsOwner, nil
}
