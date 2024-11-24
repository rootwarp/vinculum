package abi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ABI is an interface for fetching contract ABIs
type ABI interface {
	GetContractABI(ctx context.Context, address string) (ContractABIs, error)
}

type etherscanABI struct {
	apiBaseURL string
	apiKey     string
}

// GetContractABI fetches the ABI for a given contract address from the Etherscan API
func (e *etherscanABI) GetContractABI(ctx context.Context, address string) (ContractABIs, error) {
	url := fmt.Sprintf("%s/api?module=contract&action=getabi&address=%s&apikey=%s", e.apiBaseURL, address, e.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	cli := http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(content, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	if apiResp.Status != "1" || apiResp.Message != "OK" {
		return nil, fmt.Errorf("API error: %s", apiResp.Message)
	}

	var contractABIs ContractABIs
	if err := json.Unmarshal([]byte(apiResp.Result), &contractABIs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contract ABIs: %w", err)
	}

	return contractABIs, nil
}

// NewABIClient creates a new ABI client
func NewABIClient(apiBaseURL, apiKey string) ABI {
	return &etherscanABI{
		apiBaseURL: apiBaseURL,
		apiKey:     apiKey,
	}
}
