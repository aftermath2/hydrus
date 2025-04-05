package lightning_test

import (
	"testing"

	"github.com/aftermath2/hydrus/lightning"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
)

func TestParseChannelPoint(t *testing.T) {
	tests := []struct {
		name         string
		channelPoint string
		expected     *lnrpc.ChannelPoint
		fail         bool
	}{
		{
			name:         "Valid channel point",
			channelPoint: "00000000000000000000000000000001:123456789",
			expected: &lnrpc.ChannelPoint{
				FundingTxid: &lnrpc.ChannelPoint_FundingTxidStr{FundingTxidStr: "00000000000000000000000000000001"},
				OutputIndex: 123456789,
			},
			fail: false,
		},
		{
			name:         "Missing colon",
			channelPoint: "0123456789abcdef0123456789abcdef0123456789abc",
			fail:         true,
		},
		{
			name:         "Invalid output index",
			channelPoint: "0123456789abcdef0123456789abcdef0123456789abc:notANumber",
			fail:         true,
		},
		{
			name:         "Empty input",
			channelPoint: "",
			fail:         true,
		},
		{
			name:         "Leading colon",
			channelPoint: ":123456789",
			fail:         true,
		},
		{
			name:         "Trailing colon",
			channelPoint: "0123456789abcdef0123456789abcd:",
			fail:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lightning.ParseChannelPoint(tt.channelPoint)
			if tt.fail {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected.FundingTxid, got.FundingTxid)
			assert.Equal(t, tt.expected.OutputIndex, got.OutputIndex)
		})
	}
}
