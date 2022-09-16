package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	wasmutils "github.com/CosmWasm/wasmd/x/wasm/ioutils"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/types"
)

func storeCode(path string, sender types.AccAddress) (wasmtypes.MsgStoreCode, error) {
	wasm, err := ioutil.ReadFile(path)
	if err != nil {
		return wasmtypes.MsgStoreCode{}, err
	}
	if wasmutils.IsWasm(wasm) {
		wasm, err = wasmutils.GzipIt(wasm)
		if err != nil {
			return wasmtypes.MsgStoreCode{}, err
		}
	} else if !wasmutils.IsGzip(wasm) {
		return wasmtypes.MsgStoreCode{}, fmt.Errorf("invalid input file. Use wasm binary or gzip")
	}

	return wasmtypes.MsgStoreCode{
		Sender:                sender.String(),
		WASMByteCode:          wasm,
		InstantiatePermission: &wasmtypes.AllowEverybody,
	}, nil
}

func instantiateNFTContract(codeId uint64, label string, sender types.AccAddress) (wasmtypes.MsgInstantiateContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"minter": sender.String(),
		"name":   label,
		"symbol": "skronk",
	})
	if err != nil {
		return wasmtypes.MsgInstantiateContract{}, nil
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

func mintTokenMsg(contract string, sender types.AccAddress, id string, recipient types.AccAddress) (wasmtypes.MsgExecuteContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"mint": map[string]interface{}{
			"owner":    recipient.String(),
			"token_id": id,
		},
	})
	if err != nil {
		return wasmtypes.MsgExecuteContract{}, nil
	}
	return wasmtypes.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: contract,
		Msg:      msg,
		Funds:    nil,
	}, nil
}

func transferTokenMsg(contract string, sender types.AccAddress, recipient types.AccAddress, id string) (wasmtypes.MsgExecuteContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"transfer_nft": map[string]interface{}{
			"recipient": recipient.String(),
			"token_id":  id,
		},
	})
	if err != nil {
		return wasmtypes.MsgExecuteContract{}, nil
	}
	return wasmtypes.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: contract,
		Msg:      msg,
		Funds:    nil,
	}, nil
}

type nftInfoResult struct {
	Access struct {
		Owner string `json:"owner"`
	} `json:"access"`
}

type tokensResult struct {
	Tokens []string `json:"tokens"`
}

func (s *IntegrationTestSuite) TestHappyPath() {
	s.Run("Bring up chain, and test the happy path", func() {
		msg, err := storeCode("contracts/compiled/cw2981_royalties.optimized.wasm", s.chain.validators[0].keyInfo.GetAddress())
		s.Require().NoError(err)

		val := s.chain.validators[0]
		keyring, err := val.keyring()
		s.Require().NoError(err)

		clientCtx, err := s.chain.clientContext("tcp://localhost:26657", &keyring, "val", val.keyInfo.GetAddress())
		s.Require().NoError(err)

		s.T().Log("Uploading CW2981 NFT contract...")
		res, err := s.chain.sendMsgs(*clientCtx, &msg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		s.T().Log("CW721 NFT contract uploaded successfully")

		events := res.Logs[0].Events
		event := events[len(events)-1]
		attrsLen := len(event.Attributes)
		attr := event.Attributes[attrsLen-1]
		s.T().Logf("attributes: %v", event)
		s.Require().Equal("code_id", attr.Key)

		codeIdStr := attr.Value
		codeId, err := strconv.Atoi(codeIdStr)
		s.Require().NoError(err)
		s.T().Logf("Found code ID %d for CW721 NFT contract", codeId)

		s.T().Log("Instantiating NFT token contract...")
		instantiateMsg, err := instantiateNFTContract(uint64(codeId), "Skronk Token Internazionale", val.keyInfo.GetAddress())
		s.Require().NoError(err)
		res, err = s.chain.sendMsgs(*clientCtx, &instantiateMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		event = res.Logs[0].Events[0]
		s.Require().NotNil(event)
		attr = event.Attributes[0]
		s.Require().Equal("_contract_address", attr.Key)
		contractAddress := attr.Value
		s.T().Logf("NFT token contract instantiated at address: %s", contractAddress)

		for i, otherVal := range s.chain.validators {
			id := fmt.Sprintf("badonk-%d", i)
			mintMsg, err := mintTokenMsg(contractAddress, val.keyInfo.GetAddress(), id, otherVal.keyInfo.GetAddress())
			s.Require().NoError(err)
			res, err = s.chain.sendMsgs(*clientCtx, &mintMsg)
			s.Require().NoError(err)
			s.Require().Zero(res.Code)
		}

		s.T().Log("Querying contract for information about a token...")
		queryClient := wasmtypes.NewQueryClient(clientCtx)

		queryData, err := json.Marshal(map[string]interface{}{
			"all_nft_info": map[string]interface{}{
				"token_id": "badonk-0",
			},
		})
		s.Require().NoError(err)
		contractQueryData := wasmtypes.QuerySmartContractStateRequest{
			Address:   contractAddress,
			QueryData: queryData,
		}
		qres, err := queryClient.SmartContractState(context.Background(), &contractQueryData)
		s.Require().NoError(err)
		var obj nftInfoResult
		err = json.Unmarshal(qres.Data, &obj)
		s.Require().NoError(err)
		owner := obj.Access.Owner
		s.Require().Equal(val.keyInfo.GetAddress().String(), owner)
		s.T().Log("Found NFT with ID \"badonk-0\"")
		s.T().Logf("Owner id: %s", owner)

		otherVal := s.chain.validators[1]
		s.T().Logf("Sending a token from %s to %s", val.keyInfo.GetAddress().String(), otherVal.keyInfo.GetAddress().String())
		transferMsg, err := transferTokenMsg(contractAddress, val.keyInfo.GetAddress(), otherVal.keyInfo.GetAddress(), "badonk-0")
		s.Require().NoError(err)
		res, err = s.chain.sendMsgs(*clientCtx, &transferMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		s.T().Logf("Send successful, querying tokens belonging to %s", otherVal.keyInfo.GetAddress())

		queryData, err = json.Marshal(map[string]interface{}{
			"tokens": map[string]interface{}{
				"owner": otherVal.keyInfo.GetAddress(),
			},
		})
		s.Require().NoError(err)
		contractQueryData = wasmtypes.QuerySmartContractStateRequest{
			Address:   contractAddress,
			QueryData: queryData,
		}
		qres, err = queryClient.SmartContractState(context.Background(), &contractQueryData)
		s.Require().NoError(err)
		var tokens tokensResult
		err = json.Unmarshal(qres.Data, &tokens)
		s.Require().NoError(err)
		s.T().Logf("Found tokens for owner %s:", otherVal.keyInfo.GetAddress())
		for _, tokenId := range tokens.Tokens {
			s.T().Logf("- %s", tokenId)
		}
		s.Require().Equal([]string{"badonk-0", "badonk-1"}, tokens.Tokens)
	})
}
