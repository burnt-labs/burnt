package integration_tests

import (
	"context"
	"encoding/json"
	scheduletypes "github.com/BurntFinance/burnt/x/schedule/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"strconv"
	"time"
)

func instantiateTickerContract(codeId uint64, label string, sender sdktypes.AccAddress, count int32) (wasmtypes.MsgInstantiateContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"count": count,
	})
	if err != nil {
		return wasmtypes.MsgInstantiateContract{}, err
	}
	return wasmtypes.MsgInstantiateContract{
		Sender: sender.String(),
		Admin:  sender.String(),
		CodeID: codeId,
		Label:  label,
		Msg:    msg,
		Funds:  nil,
	}, nil
}

func instantiateProxyContract(codeId uint64, label string, sender sdktypes.AccAddress) (wasmtypes.MsgInstantiateContract, error) {
	msg, err := json.Marshal(map[string]interface{}{})
	if err != nil {
		return wasmtypes.MsgInstantiateContract{}, err
	}
	return wasmtypes.MsgInstantiateContract{
		Sender: sender.String(),
		Admin:  sender.String(),
		CodeID: codeId,
		Label:  label,
		Msg:    msg,
		Funds:  nil,
	}, nil
}

func tickerIncrementMsg(contract string, sender sdktypes.AccAddress) (wasmtypes.MsgExecuteContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"increment": map[string]interface{}{},
	})
	if err != nil {
		return wasmtypes.MsgExecuteContract{}, err
	}
	return wasmtypes.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: contract,
		Msg:      msg,
		Funds:    nil,
	}, nil
}

type TickerCountResponse struct {
	Count int32 `json:"count"`
}

type ProxyIsOwnerResponse struct {
	IsOwner bool `json:"is_owner"`
}

func queryIsProxyOwner(ctx *client.Context, addr string, owner sdktypes.AccAddress) (bool, error) {
	queryClient := wasmtypes.NewQueryClient(ctx)

	queryData, err := json.Marshal(map[string]interface{}{
		"is_owner": map[string]interface{}{
			"address": owner,
		},
	})
	if err != nil {
		return false, err
	}
	contractQueryData := wasmtypes.QuerySmartContractStateRequest{
		Address:   addr,
		QueryData: queryData,
	}
	qres, err := queryClient.SmartContractState(context.Background(), &contractQueryData)
	if err != nil {
		return false, err
	}
	var res ProxyIsOwnerResponse
	err = json.Unmarshal(qres.Data, &res)
	if err != nil {
		return false, err
	}

	return res.IsOwner, nil
}

func queryTickerCount(ctx *client.Context, addr string) (int, error) {
	queryClient := wasmtypes.NewQueryClient(ctx)

	queryData, err := json.Marshal(map[string]interface{}{
		"get_count": map[string]interface{}{},
	})
	if err != nil {
		return 0, err
	}
	contractQueryData := wasmtypes.QuerySmartContractStateRequest{
		Address:   addr,
		QueryData: queryData,
	}
	qres, err := queryClient.SmartContractState(context.Background(), &contractQueryData)
	if err != nil {
		return 0, err
	}
	var countResponse TickerCountResponse
	err = json.Unmarshal(qres.Data, &countResponse)
	if err != nil {
		return 0, err
	}

	return int(countResponse.Count), nil
}

func currentBlockHeight(clientCtx *client.Context) (uint64, error) {
	node, err := clientCtx.GetNode()
	if err != nil {
		return 0, err
	}
	status, err := node.Status(context.Background())
	if err != nil {
		return 0, err
	}
	return uint64(status.SyncInfo.LatestBlockHeight), nil
}

func queryScheduledCalls(clientCtx *client.Context) ([]*scheduletypes.QueryScheduledCall, error) {
	queryClient := scheduletypes.NewQueryClient(clientCtx)
	res, err := queryClient.ScheduledCalls(context.Background(), &scheduletypes.QueryScheduledCallsRequest{})
	if err != nil {
		return nil, err
	}

	return res.Calls, nil
}

func makeProxyIncrementMsg(address sdktypes.AccAddress) ([]byte, error) {
	incrementMsg, err := json.Marshal(map[string]interface{}{
		"increment": map[string]interface{}{},
	})

	proxyIncrementMsg, err := json.Marshal(map[string]interface{}{
		"destination_address": address,
		"message_to_forward":  incrementMsg,
	})
	if err != nil {
		return nil, err
	}

	return proxyIncrementMsg, nil
}

const StartCount = 1337

var CurrentCount = StartCount

func (s *IntegrationTestSuite) TestScheduledCall() {
	s.Run("Bring up chain, and test the schedule module", func() {
		uploadTickerMsg, err := storeCode("contracts/compiled/ticker.wasm", s.chain.validators[0].keyInfo.GetAddress())
		s.Require().NoError(err)

		uploadProxyMsg, err := storeCode("contracts/compiled/proxy.wasm", s.chain.validators[0].keyInfo.GetAddress())
		s.Require().NoError(err)

		val := s.chain.validators[0]
		keyring, err := val.keyring()
		s.Require().NoError(err)

		clientCtx, err := s.chain.clientContext("tcp://localhost:26657", &keyring, "val", val.keyInfo.GetAddress())
		s.Require().NoError(err)

		s.T().Log("Uploading ticker contract...")
		res, err := s.chain.sendMsgs(*clientCtx, &uploadTickerMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		s.T().Log("ticker contract uploaded successfully")

		events := res.Logs[0].Events
		event := events[len(events)-1]
		attrsLen := len(event.Attributes)
		attr := event.Attributes[attrsLen-1]
		s.Require().Equal("code_id", attr.Key)

		tickerCodeId, err := strconv.Atoi(attr.Value)
		s.Require().NoError(err)
		s.T().Logf("Found code ID %d for ticker contract", tickerCodeId)

		s.T().Log("Uploading proxy contract...")
		res, err = s.chain.sendMsgs(*clientCtx, &uploadProxyMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		s.T().Log("proxy contract uploaded successfully")

		events = res.Logs[0].Events
		event = events[len(events)-1]
		attrsLen = len(event.Attributes)
		attr = event.Attributes[attrsLen-1]
		s.Require().Equal("code_id", attr.Key)

		proxyCodeId, err := strconv.Atoi(attr.Value)
		s.Require().NoError(err)
		s.T().Logf("Found code ID %d for proxy contract", proxyCodeId)

		s.T().Log("Instantiating ticker contract...")
		instantiateMsg, err := instantiateTickerContract(uint64(tickerCodeId), "test ticker", val.keyInfo.GetAddress(), StartCount)
		s.Require().NoError(err)
		res, err = s.chain.sendMsgs(*clientCtx, &instantiateMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		event = res.Logs[0].Events[0]
		s.Require().NotNil(event)
		attr = event.Attributes[0]
		s.Require().Equal("_contract_address", attr.Key)
		tickerContractInstance, err := sdktypes.AccAddressFromBech32(attr.Value)
		s.Require().NoError(err)
		s.T().Logf("ticker contract instantiated at address: %s", tickerContractInstance.String())

		s.T().Log("Instantiating proxy contract...")
		instantiateMsg, err = instantiateProxyContract(uint64(proxyCodeId), "test proxy", val.keyInfo.GetAddress())
		s.Require().NoError(err)
		res, err = s.chain.sendMsgs(*clientCtx, &instantiateMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		event = res.Logs[0].Events[0]
		s.Require().NotNil(event)
		attr = event.Attributes[0]
		s.Require().Equal("_contract_address", attr.Key)
		proxyContractInstance, err := sdktypes.AccAddressFromBech32(attr.Value)
		s.Require().NoError(err)
		s.T().Logf("proxy contract instantiated at address: %s", proxyContractInstance.String())

		// baseline tests to make sure the contracts behave as expected
		s.T().Log("Querying ticker contract for count")
		count, err := queryTickerCount(clientCtx, tickerContractInstance.String())
		s.Require().NoError(err)
		s.Require().Equal(StartCount, count)

		s.T().Log("Querying proxy contract for owner")
		isOwner, err := queryIsProxyOwner(clientCtx, proxyContractInstance.String(), val.keyInfo.GetAddress())
		s.Require().NoError(err)
		s.Require().True(isOwner, "is_owner returned as false")

		// checking params
		scheduleQC := scheduletypes.NewQueryClient(clientCtx)
		params, err := scheduleQC.Params(context.Background(), &scheduletypes.QueryParamsRequest{})
		s.Require().NoError(err)
		s.T().Logf("params: %v", params)

		incrementMsg, err := tickerIncrementMsg(tickerContractInstance.String(), val.keyInfo.GetAddress())
		s.Require().NoError(err)
		res, err = s.chain.sendMsgs(*clientCtx, &incrementMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		CurrentCount += 1

		count, err = queryTickerCount(clientCtx, tickerContractInstance.String())
		s.Require().NoError(err)
		s.Require().Equal(CurrentCount, count)

		// query current block height
		blockHeight, err := currentBlockHeight(clientCtx)
		s.Require().NoError(err)
		s.T().Logf("current block height %d", blockHeight)

		// create the passthrough message
		proxyIncrementMsg, err := makeProxyIncrementMsg(tickerContractInstance)
		s.Require().NoError(err)

		// schedule the call, expect error because contract has no balance
		scheduledBlockHeight := blockHeight + 10
		scheduleMsg := scheduletypes.MsgAddSchedule{
			Signer:      val.keyInfo.GetAddress().String(),
			Contract:    proxyContractInstance.String(),
			CallBody:    proxyIncrementMsg,
			BlockHeight: scheduledBlockHeight,
		}
		res, err = s.chain.sendMsgs(*clientCtx, &scheduleMsg)
		s.Require().Error(err)
		s.T().Logf("failed to schedule call for height %d with %v", scheduledBlockHeight, scheduleMsg.Contract)

		s.T().Logf("transfering balance to contract")
		transferCoinMsg := banktypes.NewMsgSend(
			s.chain.validators[0].keyInfo.GetAddress(),
			proxyContractInstance,
			sdktypes.Coins{{Denom: testDenom, Amount: sdktypes.NewInt(1000000)}})

		res, err = s.chain.sendMsgs(*clientCtx, transferCoinMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)

		// schedule the call, succeed
		blockHeight, err = currentBlockHeight(clientCtx)
		s.Require().NoError(err)
		scheduledBlockHeight = blockHeight + 10
		scheduleMsg.BlockHeight = scheduledBlockHeight
		res, err = s.chain.sendMsgs(*clientCtx, &scheduleMsg)
		s.Require().NoError(err)
		s.T().Logf("scheduled call for height %d on contract %v", scheduledBlockHeight, scheduleMsg.Contract)

		// verify the call was scheduled
		s.Require().Eventuallyf(func() bool {
			height, err := currentBlockHeight(clientCtx)
			s.Require().NoError(err)
			calls, err := queryScheduledCalls(clientCtx)
			s.Require().NoError(err)
			for _, call := range calls {
				if call.Contract != proxyContractInstance.String() {
					s.T().Logf("found contract %s, expected %s", call.Contract, proxyContractInstance.String())
					continue
				} else if call.Height != scheduledBlockHeight {
					s.T().Logf("found call height %d, expected %d", call.Height, scheduledBlockHeight)
					continue
				}
				return true
			}
			s.T().Logf("queried scheduled calls at block %d, got %v", height, calls)
			return false
		}, time.Second*20, time.Second*3, "never found scheduled call")

		// watch the blocks and check if the count has updated
		s.Require().Eventuallyf(func() bool {
			blockHeight, err = currentBlockHeight(clientCtx)
			s.Require().NoError(err)
			calls, err := queryScheduledCalls(clientCtx)
			s.Require().NoError(err)
			s.T().Logf("height: %d, calls: %v", blockHeight, calls)
			if blockHeight < scheduledBlockHeight {
				count, err = queryTickerCount(clientCtx, tickerContractInstance.String())
				s.Require().NoError(err)
				s.Require().Equal(CurrentCount, count, "count was updated before schedule")
			} else if blockHeight > scheduledBlockHeight {
				count, err = queryTickerCount(clientCtx, tickerContractInstance.String())
				s.Require().NoError(err)
				s.Require().Equal(CurrentCount+1, count, "count was not updated after schedule")
				return true
			}

			return false
		}, time.Minute*1, time.Second, "never found an incremented count after the scheduled height")

		// todo: same test, but when the call returns a reschedule. validate it
		// updated, that the new height was recorded, and eventually consumed
	})
}
