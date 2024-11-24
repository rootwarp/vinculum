package contract

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/rootwarp/vinculum/contract/abi"
)

// ContractClient is an interface a contract
type ContractClient interface {
	ReadContract(ctx context.Context, addr string, abi abi.ContractABI, args map[string]interface{}) (string, error)
}

type contractClient struct {
	rpcURL string
}

func (c *contractClient) ReadContract(ctx context.Context, addr string, abi abi.ContractABI, args map[string]interface{}) (string, error) {
	if err := c.validateInputs(abi, args); err != nil {
		return "", err
	}

	data, err := c.encodeData(abi, args)
	if err != nil {
		return "", err
	}

	// Build the call data with method ID and encoded arguments
	callData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to":   addr,
				"data": data,
			},
			"latest",
		},
		"id": 1,
	}

	jsonData, err := json.Marshal(callData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal call data: %w", err)
	}

	resp, err := http.Post(c.rpcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make RPC call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	resultData := strings.TrimPrefix(result.Result, "0x")
	return c.parseResponse(resultData, abi)
}

func (c *contractClient) validateInputs(abi abi.ContractABI, args map[string]interface{}) error {
	// Check if the number of provided arguments matches the expected inputs
	if len(args) != len(abi.Inputs) {
		return fmt.Errorf("argument count mismatch: expected %d, got %d", len(abi.Inputs), len(args))
	}

	// Verify each provided argument matches the expected type
	for _, input := range abi.Inputs {
		arg, exists := args[input.Name]
		if !exists {
			return fmt.Errorf("missing argument for input %q", input.Name)
		}

		// Check if argument type matches the ABI input type
		switch input.Type {
		case "address":
			if _, ok := arg.(string); !ok {
				return fmt.Errorf("invalid type for input %q: expected address string, got %T", input.Name, arg)
			}
		case "uint256":
			if _, ok := arg.(*big.Int); !ok {
				return fmt.Errorf("invalid type for input %q: expected *big.Int, got %T", input.Name, arg)
			}
		case "bool":
			if _, ok := arg.(bool); !ok {
				return fmt.Errorf("invalid type for input %q: expected bool, got %T", input.Name, arg)
			}
		case "string":
			if _, ok := arg.(string); !ok {
				return fmt.Errorf("invalid type for input %q: expected string, got %T", input.Name, arg)
			}
		default:
			return fmt.Errorf("unsupported input type: %s", input.Type)
		}
	}

	return nil
}

func (c *contractClient) encodeData(abi abi.ContractABI, args map[string]interface{}) (string, error) {
	methodID, err := abi.MethodID()
	if err != nil {
		return "", fmt.Errorf("failed to get method ID: %w", err)
	}

	data := "0x" + methodID

	// Encode each argument according to its type and append to data
	for _, input := range abi.Inputs {
		arg := args[input.Name]
		var encoded string

		switch input.Type {
		case "address":
			// Remove "0x" prefix if present and pad address to 32 bytes
			addr := strings.TrimPrefix(arg.(string), "0x")
			encoded = fmt.Sprintf("%064s", addr)
		case "uint256":
			// Convert big.Int to hex string and pad to 32 bytes
			bigInt := arg.(*big.Int)
			encoded = fmt.Sprintf("%064s", bigInt.Text(16))
		case "bool":
			// Encode bool as 0 or 1 padded to 32 bytes
			if arg.(bool) {
				encoded = fmt.Sprintf("%064s", "1")
			} else {
				encoded = fmt.Sprintf("%064s", "0")
			}
		case "string":
			// For dynamic types like string:
			// 1. Get the string bytes
			str := []byte(arg.(string))
			// 2. Calculate offset position (32 bytes per previous static argument)
			offset := big.NewInt(int64(32 * len(abi.Inputs)))
			// 3. Add length of the string
			length := big.NewInt(int64(len(str)))
			// 4. Encode offset and length as padded hex
			encoded = fmt.Sprintf("%064s%064s", offset.Text(16), length.Text(16))
			// 5. Add the actual string data padded to nearest 32 bytes
			paddedLen := (len(str) + 31) / 32 * 32
			paddedStr := make([]byte, paddedLen)
			copy(paddedStr, str)
			encoded += hex.EncodeToString(paddedStr)
		default:
			return "", fmt.Errorf("unsupported type for encoding: %s", input.Type)
		}

		data += encoded
	}

	return data, nil
}

func (c *contractClient) parseResponse(resp string, abi abi.ContractABI) (string, error) {
	// FIXME: For now, we only handle single output parameter
	if len(abi.Outputs) != 1 {
		return "", fmt.Errorf("multiple outputs not yet supported")
	}

	output := abi.Outputs[0]
	switch output.Type {
	case "uint256":
		// Convert hex string to big.Int
		value := new(big.Int)
		value.SetString(resp, 16)
		return value.String(), nil
	case "string":
		// String data starts with offset (32 bytes), then length (32 bytes), then the actual string data
		// Skip first 64 bytes (offset + length) and decode the rest as UTF-8
		if len(resp) < 128 {
			return "", fmt.Errorf("invalid string data length")
		}
		strLen := new(big.Int)
		strLen.SetString(resp[64:128], 16)
		strData := resp[128 : 128+strLen.Int64()*2] // *2 because hex encoding
		bytes, err := hex.DecodeString(strData)
		if err != nil {
			return "", fmt.Errorf("failed to decode string data: %w", err)
		}
		return string(bytes), nil
	case "bool":
		// Bool is encoded as uint256 where 0 = false, 1 = true
		if len(resp) < 64 {
			return "", fmt.Errorf("invalid bool data length")
		}
		if resp[63] == '1' {
			return "true", nil
		}

		return "false", nil
	case "address":
		// Address is encoded as uint160
		if len(resp) < 64 {
			return "", fmt.Errorf("invalid address data length")
		}
		return "0x" + resp[24:64], nil
	case "uint8":
		// uint8 is right-padded to 32 bytes
		if len(resp) < 64 {
			return "", fmt.Errorf("invalid uint8 data length")
		}
		value := new(big.Int)
		value.SetString(resp, 16)
		if !value.IsUint64() || value.Uint64() > 255 {
			return "", fmt.Errorf("uint8 overflow")
		}
		return value.String(), nil
	default:
		return "", fmt.Errorf("unsupported output type: %s", output.Type)
	}
}

// NewClient creates a new contract client
func NewClient(rpcURL string) ContractClient {
	return &contractClient{
		rpcURL: rpcURL,
	}
}
