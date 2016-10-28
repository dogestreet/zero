package stratum

import (
	"encoding/json"
	"errors"
	"log"
	"math/big"

	"github.com/hashicorp/errwrap"
)

// For clarity, a "Request" is defined as a message sent from the client to the server.
// A "Response" is defined as a message sent by the server to the client.
type (
	// RequestType ...
	RequestType string

	// ResponseType ...
	ResponseType string

	// An Uint128 is a 128-bit unsigned integer.
	// Internal representation is big endian.
	Uint128 [16]uint8

	// An Uint256 is a 256bit unsigned integer.
	// Internal representation is big endian.
	Uint256 [32]uint8

	// Request ...
	Request interface {
		Type() RequestType
	}

	// Response ...
	Response interface {
		Type() ResponseType
	}
)

// Request and response definitions.
type (
	// RawRPC ...
	RawRPC struct {
		ID     interface{}      `json:"id"`
		Method RequestType      `json:"method"`
		Params *json.RawMessage `json:"params"`
	}

	// RequestBase are fields present in every request.
	RequestBase struct {
		ID     interface{} `json:"id"`
		Method RequestType `json:"method"`
	}

	// RequestSubscribe is sent by the client when during the initial connect.
	RequestSubscribe struct {
		RequestBase

		Params []string
	}

	// RequestAuthorise ...
	RequestAuthorise struct {
		RequestBase

		Username string
		Password string
	}

	// RequestSubmit is sent by the client when submitting a share.
	RequestSubmit struct {
		RequestBase

		Worker string
		Job    string
		NTime  uint32

		NoncePart2 Uint128
		Solution   []byte
	}

	// RequestSuggestTarget is sent by the client to suggest a target.
	RequestSuggestTarget struct {
		RequestBase

		Target Uint256
	}

	// ResponseSubscribeReply is sent in reply to the subscription request.
	ResponseSubscribeReply struct {
		ID         interface{}
		Session    string
		NoncePart1 Uint128
	}

	// zcash has 256 bit block nonces
	// We can hold back much as much as we want, but we want to maximise the amount the miners get.
	// Therefore we divide the nonce according to the following scheme:
	// [Server UID][Client ID][ Leftover        ]
	// [ 64-bits  ][ 64-bits ][ 128-bits        ]
	// This allows us to pick server IDs randomly, with a 2^128 collision probability.

	// ResponseNotify contains the fields required to construct the block header.
	// See: https://github.com/zcash/zcash/blob/70db019c6ae989acde0a0affd6a1f1c28ec9a3d2/src/primitives/block.h#L23
	ResponseNotify struct {
		Job            string
		Version        uint32 // Block header version 4
		HashPrevBlock  Uint256
		HashMerkleRoot Uint256
		HashReserved   Uint256
		NTime          uint32
		NBits          uint32
		CleanJobs      bool
	}

	// ResponseSetTarget sets the difficulty on the client.
	ResponseSetTarget struct {
		Target Uint256 // Another difference to bitcoin, the raw value for the difficulty is used
	}

	// ResponseGeneral is a general response.
	ResponseGeneral struct {
		ID interface{}

		Result interface{}
		Error  interface{}
	}
)

// ErrUnknownType is returns when the "method" of the JSON request isn't a known stratum request type.
var ErrUnknownType = errors.New("unknown type")

// ErrBadInput is returned when bad input from the client is detected.
var ErrBadInput = errors.New("bad input")

// RequestType definitions.
const (
	Subscribe     RequestType = "mining.subscribe"
	Authorise     RequestType = "mining.authorize"
	Submit        RequestType = "mining.submit"
	SuggestTarget RequestType = "mining.suggest_target"
)

// ResponseType definitions.
const (
	SubscribeReply ResponseType = "mining.subscribe.reply"
	Notify         ResponseType = "mining.notify"
	SetTarget      ResponseType = "mining.set_target"
	General        ResponseType = "general"
)

func marshalRequest(request RawRPC, params interface{}) ([]byte, error) {
	result, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	request.Params = (*json.RawMessage)(&result)
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Type implements Request.
func (r RequestBase) Type() RequestType {
	return r.Method
}

// MarshalJSON implementation for the Request object.
func (r RequestSubscribe) MarshalJSON() ([]byte, error) {
	return marshalRequest(RawRPC{
		ID:     r.ID,
		Method: Subscribe,
	}, make([]int, 0))
}

// MarshalJSON implementation for the Request object.
func (r RequestAuthorise) MarshalJSON() ([]byte, error) {
	return marshalRequest(RawRPC{
		ID:     r.ID,
		Method: Authorise,
	}, []string{r.Username, r.Password})
}

// MarshalJSON implementation for the Request object.
func (r RequestSubmit) MarshalJSON() ([]byte, error) {
	return marshalRequest(RawRPC{
		ID:     r.ID,
		Method: Submit,
	}, []interface{}{r.Worker, r.Job, r.NTime, r.NoncePart2})
}

// Parse bytes into a request object.
func Parse(data []byte) (Request, error) {
	var raw RawRPC
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	base := RequestBase{
		ID:     raw.ID,
		Method: raw.Method,
	}

	switch RequestType(raw.Method) {
	case Subscribe:
		return RequestSubscribe{
			RequestBase: base,
			Params:      nil,
		}, nil

	case Authorise:
		var params [2]string
		if err := json.Unmarshal(*raw.Params, &params); err != nil {
			log.Println(err)
			return nil, errwrap.Wrapf("error decoding auth params: {{err}}", ErrBadInput)
		}

		return RequestAuthorise{
			RequestBase: base,
			Username:    params[0],
			Password:    params[1],
		}, nil

	case Submit:
		var params [5]string
		if err := json.Unmarshal(*raw.Params, &params); err != nil {
			return nil, errwrap.Wrapf("error decoding submit params: {{err}}", ErrBadInput)
		}

		ntime, err := HexToUint32(params[2])
		if err != nil {
			return nil, ErrBadInput
		}

		nonce, err := HexToUint128(params[3])
		if err != nil {
			return nil, ErrBadInput
		}

		// The solution includes the 3 bytes of the solution size.
		solution, err := readHex(params[4], 1347)
		if err != nil {
			return nil, ErrBadInput
		}

		// Remove the compact size
		solution = solution[3:]

		return RequestSubmit{
			RequestBase: base,
			Worker:      params[0],
			Job:         params[1],
			NTime:       ntime,
			NoncePart2:  nonce,
			Solution:    solution,
		}, nil

	case SuggestTarget:
		var params [1]string
		if err := json.Unmarshal(*raw.Params, &params); err != nil {
			return nil, errwrap.Wrapf("error decoding suggest_target params: {{err}}", ErrBadInput)
		}

		target, err := HexToUint256(params[0])
		if err != nil {
			return nil, ErrBadInput
		}

		return RequestSuggestTarget{
			RequestBase: base,
			Target:      target,
		}, nil

	default:
		return nil, ErrUnknownType
	}
}

// MarshalJSON implementation for the Subscribe response.
func (r ResponseSubscribeReply) MarshalJSON() ([]byte, error) {
	return json.Marshal(ResponseGeneral{
		ID:     r.ID,
		Result: []interface{}{r.Session, ToHex(r.NoncePart1)},
	})
}

// Type implements Response.
func (r ResponseSubscribeReply) Type() ResponseType {
	return SubscribeReply
}

// MarshalJSON implementation for the Notify response.
func (r ResponseNotify) MarshalJSON() ([]byte, error) {
	return marshalRequest(RawRPC{
		ID:     nil,
		Method: RequestType(Notify),
	}, []interface{}{
		r.Job,
		ToHex(r.Version),
		ToHex(r.HashPrevBlock),
		ToHex(r.HashMerkleRoot),
		ToHex(r.HashReserved),
		ToHex(r.NTime),
		ToHex(r.NBits),
		r.CleanJobs,
	})
}

// Type implements Response.
func (r ResponseNotify) Type() ResponseType {
	return Notify
}

// MarshalJSON implementation for the SetTarget response.
func (r ResponseSetTarget) MarshalJSON() ([]byte, error) {
	return marshalRequest(RawRPC{
		ID:     nil,
		Method: RequestType(SetTarget),
	}, []interface{}{
		ToHex(r.Target),
	})
}

// Type implements Response.
func (r ResponseSetTarget) Type() ResponseType {
	return SetTarget
}

// MarshalJSON implementation for the General response.
func (r ResponseGeneral) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":     r.ID,
		"result": r.Result,
		"error":  r.Error,
	})
}

// Type implements Response.
func (r ResponseGeneral) Type() ResponseType {
	return General
}

// ToInteger converts a Uint256 to a big.Int
func (uint256 Uint256) ToInteger() *big.Int {
	x := big.NewInt(0)
	return x.SetBytes(uint256[:])
}
