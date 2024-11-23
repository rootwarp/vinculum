package abi

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
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
