package channel

import (
	"encoding/hex"
	"testing"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestManagerOpen(t *testing.T) {
	ctx := t.Context()
	config := config.ChannelManager{
		MinConf:     2,
		BaseFeeMsat: 1,
		FeeRatePPM:  100,
	}
	publicKey := "025602698dc2a8fc3146cb2bb284d0768feb41390fbee4c6a72628195b39f50349"
	publicKeyB, err := hex.DecodeString(publicKey)
	assert.NoError(t, err)
	req := OpenRequest{
		Nodes: map[string]uint64{
			publicKey: 1_000_000,
		},
		SatvB: 2,
	}

	lndMock := lightning.NewClientMock()
	batchReq := &lnrpc.BatchOpenChannelRequest{
		Channels: []*lnrpc.BatchOpenChannel{
			{
				NodePubkey:         publicKeyB,
				LocalFundingAmount: 1_000_000,
				BaseFee:            config.BaseFeeMsat,
				UseBaseFee:         true,
				FeeRate:            config.FeeRatePPM,
				UseFeeRate:         true,
			},
		},
		MinConfs:              config.MinConf,
		SatPerVbyte:           int64(req.SatvB),
		SpendUnconfirmed:      false,
		Label:                 "Hydrus",
		CoinSelectionStrategy: lnrpc.CoinSelectionStrategy_STRATEGY_USE_GLOBAL_CONFIG,
	}
	lndMock.On("BatchOpenChannel", ctx, batchReq).Return("1", nil)

	manager := NewManager(config, lndMock)

	err = manager.Open(ctx, req)
	assert.NoError(t, err)
}

func TestManagerClose(t *testing.T) {
	tests := []struct {
		desc  string
		force bool
	}{
		{
			desc:  "Mutual close",
			force: false,
		},
		{
			desc:  "Force close",
			force: true,
		},
	}

	ctx := t.Context()
	channelPoint := "e5b8ccc43b4eea6e2664a843e27d82c6d71d2885e7aef73777dd35c737c1d7bc:1"
	chanPoint, err := parseChannelPoint(channelPoint)
	assert.NoError(t, err)

	config := config.ChannelManager{MaxSatvB: 10}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			req := CloseRequest{
				Channels: map[string]bool{
					channelPoint: tt.force,
				},
				SatvB: 5,
			}
			closeReq := &lnrpc.CloseChannelRequest{
				ChannelPoint:   chanPoint,
				SatPerVbyte:    req.SatvB,
				MaxFeePerVbyte: config.MaxSatvB,
				Force:          tt.force,
			}

			lndMock := lightning.NewClientMock()
			lndMock.On("CloseChannel", mock.Anything, closeReq).Return(&mockStream{}, nil)

			manager := NewManager(config, lndMock)

			err = manager.Close(ctx, req)
			assert.NoError(t, err)
		})
	}
}

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
			got, err := parseChannelPoint(tt.channelPoint)
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

type mockStream struct{}

func (m *mockStream) Recv() (*lnrpc.CloseStatusUpdate, error) {
	txID, err := hex.DecodeString("6c22520d81df34013b072a4aaf3cb858ae41c1ed9870fd3c04471d428fe11a88")
	if err != nil {
		return nil, err
	}

	return &lnrpc.CloseStatusUpdate{
		Update: &lnrpc.CloseStatusUpdate_ClosePending{
			ClosePending: &lnrpc.PendingUpdate{
				Txid:        txID,
				OutputIndex: 1,
			},
		},
	}, nil
}
