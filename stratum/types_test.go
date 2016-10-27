package stratum

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	// TODO:
	table := []struct {
		raw    string
		expect Request
	}{
		// Subscribe
		{
			raw: `{"id":1,"method":"mining.subscribe", "params":[]}`,
			expect: RequestSubscribe{
				RequestBase: RequestBase{
					ID:     1,
					Method: "mining.subscribe",
				},
			},
		},
		{
			raw: `{"id":"1","method":"mining.subscribe", "params":[]}`,
			expect: RequestSubscribe{
				RequestBase: RequestBase{
					ID:     "1",
					Method: "mining.subscribe",
				},
			},
		},

		// Authorise
		{
			raw: `{"id": 2, "method": "mining.authorize", "params": ["WORKER_NAME", "WORKER_PASSWORD"]}`,
			expect: RequestAuthorise{
				RequestBase: RequestBase{
					ID:     2,
					Method: "mining.authorize",
				},
				Username: "WORKER_NAME",
				Password: "WORKER_PASSWORD",
			},
		},

		// Submit
		// TODO:

		// SuggestTarget
		{
			raw: `{"id": 3, "method": "mining.suggest_target", "params": ["000005dda6000000000000000000000000000000000000000000000000000000"]}`,
			expect: RequestSuggestTarget{
				RequestBase: RequestBase{
					ID:     3,
					Method: "mining.suggest_target",
				},
				Target: Uint256{0x00, 0x00, 0x05, 0xdd, 0xa6},
			},
		},
	}

	for _, v := range table {
		result, err := Parse([]byte(v.raw))
		if err != nil {
			if v.expect == nil {
				continue
			}

			t.Fatal("bad parse", err)
		}

		// We can't DeepEqual here because they are both wrapped in interfaces.
		if fmt.Sprintf("%#v", result) != fmt.Sprintf("%#v", v.expect) {
			t.Fatal("bad parse", fmt.Sprintf("%#v", result), fmt.Sprintf("%#v", v.expect))
		}
	}
}

func TestMarshal(t *testing.T) {
	// TODO:
	table := []struct {
		raw    interface{}
		expect string
	}{
		// Notify
		// TODO:

		// SubscribeReply
		{
			raw: ResponseSubscribeReply{
				ID:         1,
				Session:    "abc",
				NoncePart1: Uint128{0x11, 0x22, 0x33, 0x44, 0x55},
			},
			expect: `{"error":null,"id":1,"result":["abc","11223344550000000000000000000000"]}`,
		},
	}

	for _, v := range table {
		result, err := json.Marshal(v.raw)
		if err != nil {
			if v.expect == "" {
				continue
			}

			t.Fatal("bad marshal", err)
		}

		if string(result) != v.expect {
			t.Fatal("bad marshal", string(result), v.expect)
		}
	}
}
