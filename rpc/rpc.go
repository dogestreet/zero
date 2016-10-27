package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type (
	// Client to the z.cash daemon.
	Client struct {
		address url.URL
	}

	rpcCall struct {
		Method string        `json:"method"`
		ID     *string       `json:"id"`
		Params []interface{} `json:"params"`
	}

	rpcResult struct {
		Result *json.RawMessage       `json:"result"`
		Error  map[string]interface{} `json:"error"`
		ID     *string                `json:"id"`
	}

	rpcResultError map[string]interface{}
)

// Error implements the error interface for rpcResultError.
func (rpe rpcResultError) Error() string {
	return fmt.Sprint(map[string]interface{}(rpe))
}

// NewClient creates a new JSON-RPC client.
func NewClient(address string) (*Client, error) {
	url, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	return &Client{
		address: *url,
	}, nil
}

// Call a RPC function.
func (c *Client) Call(method string, params []interface{}, result interface{}) error {
	// Generate the request
	data, err := json.Marshal(rpcCall{
		Method: method,
		ID:     nil,
		Params: params,
	})
	if err != nil {
		return err
	}

	// Post the request.
	resp, err := http.Post(c.address.String(), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode the result
	var rpcResult rpcResult
	if err := json.NewDecoder(resp.Body).Decode(&rpcResult); err != nil {
		return err
	}

	// Check for errors
	if rpcResult.Error != nil {
		return rpcResultError(rpcResult.Error)
	}

	// Check if there is a result
	if rpcResult.Result != nil {
		// Decode the data inside.
		if err := json.Unmarshal([]byte(*rpcResult.Result), result); err != nil {
			return err
		}
	}

	return nil
}
