package local_test

import (
	"errors"
	"testing"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/heuristic"
	"github.com/aftermath2/hydrus/lightning"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetNode(t *testing.T) {
	tests := []struct {
		desc            string
		balance         int64
		maxOpenChannels uint64
	}{
		{
			desc:            "Enough balance",
			balance:         15_000_000,
			maxOpenChannels: 2,
		},
		{
			desc:            "Not enough balance",
			balance:         1_000_000,
			maxOpenChannels: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := t.Context()
			config := config.Agent{
				AllocationPercent: 100,
				MinChannels:       2,
				MaxChannels:       10,
				MinChannelSize:    1_000_000,
				TargetConf:        2,
				ChannelManager: config.ChannelManager{
					MinConf: 2,
				},
				HeuristicWeights: config.HeuristicsWeights{
					Close: config.DefaultCloseWeights,
					Open:  config.DefaultOpenWeights,
				},
			}

			infoResp := &lnrpc.GetInfoResponse{
				IdentityPubkey:      "test",
				SyncedToGraph:       true,
				NumActiveChannels:   5,
				NumPendingChannels:  2,
				NumInactiveChannels: 1,
				BlockHeight:         200,
			}
			walletResp := &lnrpc.WalletBalanceResponse{
				ConfirmedBalance: tt.balance,
			}
			channelID := uint64(191315023298560)
			channelsResp := []*lnrpc.Channel{
				{
					Active:       true,
					RemotePubkey: "test_peer",
					ChannelPoint: "e5b8ccc43b4eea6e2664a843e27d82c6d71d2885e7aef73777dd35c737c1d7bc:1",
					ChanId:       channelID,
					Capacity:     2_000_000,
				},
			}
			peersResp := []*lnrpc.Peer{
				{
					PubKey:    channelsResp[0].RemotePubkey,
					Address:   "172.18.0.2:9735",
					FlapCount: 1,
					PingTime:  300,
				},
			}
			closedChannelsResp := []*lnrpc.ChannelCloseSummary{}
			feeResp := uint64(1)
			forwardsResp := &lnrpc.ForwardingHistoryResponse{
				ForwardingEvents: []*lnrpc.ForwardingEvent{
					{
						ChanIdIn:  channelID,
						AmtInMsat: 500,
						FeeMsat:   10,
					},
				},
			}
			numChannels := uint64(infoResp.NumActiveChannels + infoResp.NumInactiveChannels + infoResp.NumPendingChannels)

			lndMock := lightning.NewClientMock()
			lndMock.On("GetInfo", ctx).Return(infoResp, nil)
			lndMock.On("WalletBalance", ctx, config.ChannelManager.MinConf).Return(walletResp, nil)
			lndMock.On("ListChannels", ctx).Return(channelsResp, nil)
			lndMock.On("ListPeers", ctx).Return(peersResp, nil)
			lndMock.On("ClosedChannels", ctx).Return(closedChannelsResp, nil)
			lndMock.On("EstimateTxFee", ctx, config.TargetConf).Return(feeResp, nil)
			lndMock.On("ListForwards", ctx, uint64(0), mock.Anything, mock.Anything, uint32(0)).Return(forwardsResp, nil)

			expectedNode := local.Node{
				PublicKey: infoResp.IdentityPubkey,
				ChannelPeers: map[string]struct{}{
					channelsResp[0].RemotePubkey: {},
				},
				SyncPeers: map[string]struct{}{
					peersResp[0].PubKey: {},
				},
				ClosedChannels:     closedChannelsResp,
				AllocatedBalance:   uint64(walletResp.ConfirmedBalance),
				NumChannels:        numChannels,
				MaxOpenChannels:    tt.maxOpenChannels,
				MaxCloseChannels:   numChannels - config.MinChannels,
				SatvB:              feeResp,
				CurrentBlockHeight: infoResp.BlockHeight,
				Channels: local.Channels{
					List: []local.Channel{
						{
							Active:          channelsResp[0].Active,
							RemotePublicKey: channelsResp[0].RemotePubkey,
							Point:           channelsResp[0].ChannelPoint,
							ID:              channelsResp[0].ChanId,
							BlockHeight:     174,
							Capacity:        uint64(channelsResp[0].Capacity),
							NumForwards:     1,
							ForwardsAmount:  500,
							Fees:            10,
							FlapCount:       peersResp[0].FlapCount,
							PingTime:        peersResp[0].PingTime,
						},
					},
					Heuristics: local.Heuristics{
						Active:         heuristic.NewFull(0, 1, config.HeuristicWeights.Close.Active, false),
						Capacity:       heuristic.NewFull[uint64](2_000_000, 2_000_000, config.HeuristicWeights.Close.Capacity, false),
						NumForwards:    heuristic.NewFull[uint64](1, 1, config.HeuristicWeights.Close.NumForwards, false),
						ForwardsAmount: heuristic.NewFull[uint64](500, 500, config.HeuristicWeights.Close.ForwardsAmount, false),
						Fees:           heuristic.NewFull[uint64](10, 10, config.HeuristicWeights.Close.Fees, false),
						BlockHeight:    heuristic.NewFull[uint64](174, 174, config.HeuristicWeights.Close.BlockHeight, true),
						PingTime:       heuristic.NewFull[uint64](300, 300, config.HeuristicWeights.Close.PingTime, true),
						FlapCount:      heuristic.NewFull[uint64](1, 1, config.HeuristicWeights.Close.FlapCount, true),
					},
				},
			}

			node, err := local.GetNode(ctx, config, lndMock)
			assert.NoError(t, err)

			assert.Equal(t, expectedNode, node)
		})
	}
}

func TestGetNodeErrors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		desc    string
		lndMock func() lightning.Client
	}{
		{
			desc: "RPC call error",
			lndMock: func() lightning.Client {
				lndMock := lightning.NewClientMock()
				lndMock.On("GetInfo", ctx).Return(nil, errors.New("error"))
				return lndMock
			},
		},
		{
			desc: "Not synced to graph",
			lndMock: func() lightning.Client {
				infoResp := &lnrpc.GetInfoResponse{SyncedToGraph: false}
				lndMock := lightning.NewClientMock()
				lndMock.On("GetInfo", ctx).Return(infoResp, nil)
				return lndMock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := local.GetNode(ctx, config.Agent{}, tt.lndMock())
			assert.Error(t, err)
		})
	}
}
