package abi

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

/*
In the context of Ethereum’s Application Binary Interface (ABI), the stateMutability attribute specifies a function’s interaction with the blockchain state and its ability to handle Ether. This attribute informs users and tools about the function’s behavior, aiding in appropriate interaction. The possible values for stateMutability are:
	•	pure: Indicates that the function neither reads from nor writes to the blockchain state. It operates solely on its input parameters and does not access any state variables.
	•	view: Signifies that the function reads from the blockchain state but does not modify it. Such functions can access state variables but cannot alter them.
	•	nonpayable: Denotes that the function may modify the blockchain state but does not accept Ether. Attempting to send Ether to this function will result in a transaction failure.
	•	payable: Indicates that the function can modify the blockchain state and is capable of receiving Ether. This is essential for functions intended to handle Ether transfers.
*/

// APIResponse represents the top-level response from the API
type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"` // This is a JSON string that needs to be parsed separately
}

// ContractABI represents a single ABI entry after parsing the Result field
type ContractABI struct {
	Constant        bool           `json:"constant"`
	Inputs          []ABIParameter `json:"inputs"`
	Name            string         `json:"name,omitempty"`
	Outputs         []ABIParameter `json:"outputs"`
	Payable         bool           `json:"payable"`
	StateMutability string         `json:"stateMutability"`
	Type            string         `json:"type"`
	Anonymous       bool           `json:"anonymous,omitempty"` // Only for events
	Indexed         bool           `json:"indexed,omitempty"`   // Only for event parameters
}

// MethodID returns the first 4 bytes of the Keccak256 hash of the function signature as a hex string.
// For functions, the signature is constructed as name(type1,type2,...).
// Returns an error if the ABI entry is not a function or if the signature cannot be constructed.
func (c *ContractABI) MethodID() (string, error) {
	if c.Type != "function" {
		return "", fmt.Errorf("cannot get method ID for non-function type: %s", c.Type)
	}

	var inputTypes []string
	for _, input := range c.Inputs {
		inputTypes = append(inputTypes, input.Type)
	}
	signature := fmt.Sprintf("%s(%s)", c.Name, strings.Join(inputTypes, ","))

	hash := crypto.Keccak256([]byte(signature))
	return hex.EncodeToString(hash[:4]), nil
}

// ABIParameter represents an input or output parameter in the ABI
type ABIParameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed,omitempty"` // Only used for event parameters
}

type ContractABIs []ContractABI

// Find returns the first ContractABI with the given name, or nil if not found
func (l ContractABIs) Find(name string) (*ContractABI, error) {
	for i := range l {
		if l[i].Name == name {
			return &l[i], nil
		}
	}
	return nil, fmt.Errorf("contract ABI with name %q not found", name)
}
