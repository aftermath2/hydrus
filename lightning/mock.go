package lightning

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/stretchr/testify/mock"
)

// BlockedStreamMock is a stream of events that blocks to let the test execution end without failures.
type BlockedStreamMock[T any] struct{}

// Recv blocks forever.
func (s BlockedStreamMock[T]) Recv() (T, error) {
	// Block execution to let tests run
	block := make(chan struct{})
	<-block
	var v T
	return v, nil
}

// ClientMock is a mocked implementation of a lightning client.
type ClientMock struct {
	mock.Mock
}

// NewClientMock returns a mocked lightning network node client.
func NewClientMock() *ClientMock {
	return &ClientMock{}
}

// BatchOpenChannel mock.
func (c *ClientMock) BatchOpenChannel(ctx context.Context, req *lnrpc.BatchOpenChannelRequest) (string, error) {
	args := c.Called(ctx, req)
	return mockReturn[string](args)
}

// CloseChannel mock.
func (c *ClientMock) CloseChannel(ctx context.Context, req *lnrpc.CloseChannelRequest) (Stream[*lnrpc.CloseStatusUpdate], error) {
	args := c.Called(ctx, req)
	return mockReturn[Stream[*lnrpc.CloseStatusUpdate]](args)
}

// ClosedChannels mock.
func (c *ClientMock) ClosedChannels(ctx context.Context) ([]*lnrpc.ChannelCloseSummary, error) {
	args := c.Called(ctx)
	return mockReturn[[]*lnrpc.ChannelCloseSummary](args)
}

// ConnectPeer mock.
func (c *ClientMock) ConnectPeer(ctx context.Context, publicKey string, addresses []string) error {
	args := c.Called(ctx, publicKey, addresses)
	return args.Error(0)
}

// DescribeGraph mock.
func (c *ClientMock) DescribeGraph(ctx context.Context) (*lnrpc.ChannelGraph, error) {
	args := c.Called(ctx)
	return mockReturn[*lnrpc.ChannelGraph](args)
}

// EstimateTxFee mock.
func (c *ClientMock) EstimateTxFee(ctx context.Context, targetConf int32) (uint64, error) {
	args := c.Called(ctx, targetConf)
	return mockReturn[uint64](args)
}

// EstimateRouteFee mock.
func (c *ClientMock) EstimateRouteFee(ctx context.Context, publicKey string) (*routerrpc.RouteFeeResponse, error) {
	args := c.Called(ctx, publicKey)
	return mockReturn[*routerrpc.RouteFeeResponse](args)
}

// GetChanInfo mock.
func (c *ClientMock) GetChanInfo(ctx context.Context, channelID uint64) (*lnrpc.ChannelEdge, error) {
	args := c.Called(ctx, channelID)
	return mockReturn[*lnrpc.ChannelEdge](args)
}

// GetInfo mock.
func (c *ClientMock) GetInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error) {
	args := c.Called(ctx)
	return mockReturn[*lnrpc.GetInfoResponse](args)
}

// ListChannels mock.
func (c *ClientMock) ListChannels(ctx context.Context) ([]*lnrpc.Channel, error) {
	args := c.Called(ctx)
	return mockReturn[[]*lnrpc.Channel](args)
}

// ListForwards mock.
func (c *ClientMock) ListForwards(
	ctx context.Context,
	channelID uint64,
	startTime,
	endTime uint64,
	indexOffset uint32,
) (*lnrpc.ForwardingHistoryResponse, error) {
	args := c.Called(ctx, channelID, startTime, endTime, indexOffset)
	return mockReturn[*lnrpc.ForwardingHistoryResponse](args)
}

// ListPeers mock.
func (c *ClientMock) ListPeers(ctx context.Context) ([]*lnrpc.Peer, error) {
	args := c.Called(ctx)
	return mockReturn[[]*lnrpc.Peer](args)
}

// QueryRoute mock.
func (c *ClientMock) QueryRoute(ctx context.Context, publicKey string) (*lnrpc.QueryRoutesResponse, error) {
	args := c.Called(ctx, publicKey)
	return mockReturn[*lnrpc.QueryRoutesResponse](args)
}

// WalletBalance mock.
func (c *ClientMock) WalletBalance(ctx context.Context, minConf int32) (*lnrpc.WalletBalanceResponse, error) {
	args := c.Called(ctx, minConf)
	return mockReturn[*lnrpc.WalletBalanceResponse](args)
}

// UpdateChannelPolicy mock.
func (c *ClientMock) UpdateChannelPolicy(
	ctx context.Context, channelPoint string, baseFeeMsat, feeRatePPM, maxHTLCMsat, timeLockDelta uint64,
) error {
	args := c.Called(ctx, channelPoint, baseFeeMsat, feeRatePPM, maxHTLCMsat, timeLockDelta)
	return args.Error(0)
}

func mockReturn[T any](args mock.Arguments) (T, error) {
	var r0 T
	v0 := args.Get(0)
	if v0 != nil {
		r0 = v0.(T)
	}
	return r0, args.Error(1)
}
