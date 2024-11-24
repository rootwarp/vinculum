package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/rootwarp/vinculum/contract/abi"
	"github.com/stretchr/testify/require"
)

func TestContract_Read(t *testing.T) {
	// TODO: Mock test
	d, err := os.ReadFile("./abi/fixtures/resp_get_contract_abi.json")
	require.NoError(t, err)

	var apiResp abi.APIResponse
	err = json.Unmarshal(d, &apiResp)
	require.NoError(t, err)

	var contractABIs abi.ContractABIs
	err = json.Unmarshal([]byte(apiResp.Result), &contractABIs)
	require.NoError(t, err)

	totalSupply, err := contractABIs.Find("totalSupply")
	require.NoError(t, err)

	cli := NewClient("https://polygon-rpc.com")

	ctx := context.Background()
	contractAddr := "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270"
	ret, err := cli.ReadContract(ctx, contractAddr, *totalSupply, map[string]interface{}{})

	fmt.Println(ret, err)
}
