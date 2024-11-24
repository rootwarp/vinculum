package abi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbi_Parse(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Read mock response from fixture file
	mockRespBody, err := os.ReadFile("fixtures/resp_get_contract_abi.json")
	assert.NoError(t, err)

	// Register mock response
	httpmock.RegisterResponder(
		http.MethodGet,
		`=~^https://api\.polygonscan\.com/api\?module=contract&action=getabi&address=`,
		httpmock.NewStringResponder(http.StatusOK, string(mockRespBody)))

	abiClient := NewABIClient("https://api.polygonscan.com", "DUMMY_API_KEY")

	// Parse the result string into ContractABI slice
	contractABIs, err := abiClient.GetContractABI(context.Background(), "CONTRACT_ADDRESS")
	assert.NoError(t, err)
	// Verify we got the expected number of ABI entries
	assert.Len(t, contractABIs, 16)

	// Test a few specific ABI entries
	assert.Equal(t, "name", contractABIs[0].Name)
	assert.True(t, contractABIs[0].Constant)
	assert.Equal(t, "function", contractABIs[0].Type)
	assert.Equal(t, "view", contractABIs[0].StateMutability)
	assert.Empty(t, contractABIs[0].Inputs)
	assert.Len(t, contractABIs[0].Outputs, 1)
	assert.Equal(t, "string", contractABIs[0].Outputs[0].Type)

	// Test the "Transfer" event
	transferEvent := contractABIs[13]
	assert.Equal(t, "Transfer", transferEvent.Name)
	assert.Equal(t, "event", transferEvent.Type)
	assert.False(t, transferEvent.Anonymous)
	assert.Len(t, transferEvent.Inputs, 3)
	assert.True(t, transferEvent.Inputs[0].Indexed)
	assert.Equal(t, "src", transferEvent.Inputs[0].Name)
	assert.Equal(t, "address", transferEvent.Inputs[0].Type)
}

func TestAbi_MethodID(t *testing.T) {
	// Read mock response from fixture file
	d, err := os.ReadFile("fixtures/resp_get_contract_abi.json")
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(d, &apiResp)
	require.NoError(t, err)

	var contractABIs ContractABIs
	err = json.Unmarshal([]byte(apiResp.Result), &contractABIs)
	require.NoError(t, err)

	// ref. https://polygonscan.com/address/0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270#writeContract
	approveFunc, err := contractABIs.Find("approve")
	require.NoError(t, err)

	methodID, err := approveFunc.MethodID()
	require.NoError(t, err)
	assert.Equal(t, "095ea7b3", methodID)

	transferFromFunc, err := contractABIs.Find("transferFrom")
	require.NoError(t, err)

	methodID, err = transferFromFunc.MethodID()
	require.NoError(t, err)
	assert.Equal(t, "23b872dd", methodID)
}

func TestAbi_ReadContract(t *testing.T) {
	// polygon
	// contract address: 0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270
	// read totalSupply
	// Call eth_call
	// https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_call
	// Often used for executing read-only smart contract functions, for example the balanceOf for an ERC-20 contract.
	// Create function signature for totalSupply()
	funcSig := []byte("totalSupply()")
	hash := crypto.Keccak256(funcSig)
	methodID := hex.EncodeToString(hash[:4])

	// Create request body for eth_call
	callData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to":   "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270",
				"data": "0x" + methodID, // First 4 bytes of keccak256(totalSupply())
			},
			"latest",
		},
		"id": 1,
	}

	t.Logf("eth_call request body: %+v", callData)

	//

	rpc := "https://polygon-rpc.com"

	jsonData, err := json.Marshal(callData)
	require.NoError(t, err)

	resp, err := http.Post(rpc, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("eth_call response: %s", string(body))

	var result struct {
		Result string `json:"result"`
	}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	// Remove "0x" prefix if present
	hexStr := strings.TrimPrefix(result.Result, "0x")

	// Convert hex string to big.Int
	totalSupply := new(big.Int)
	totalSupply.SetString(hexStr, 16)

	t.Logf("Total supply: %s", totalSupply.String())

	// Create request body for eth_call with balanceOf
	methodID = hex.EncodeToString(crypto.Keccak256([]byte("balanceOf(address)"))[:4])
	t.Logf("balanceOf method ID: %x", methodID)

	balanceOfCallData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to": "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270",
				"data": "0x" + methodID +
					// Example address parameter padded to 32 bytes
					"000000000000000000000000" + "17f935d9b5E73C63b1CeC73f97dD988c5E2D9214",
			},
			"latest",
		},
		"id": 1,
	}

	t.Logf("balanceOf eth_call request body: %+v", balanceOfCallData)

	jsonData, err = json.Marshal(balanceOfCallData)
	require.NoError(t, err)

	resp, err = http.Post(rpc, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("balanceOf eth_call response: %s", string(body))
}
