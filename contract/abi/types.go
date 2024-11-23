package abi

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

// ABIParameter represents an input or output parameter in the ABI
type ABIParameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed,omitempty"` // Only used for event parameters
}
