package lightning

import (
	"context"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/logger"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/autopilotrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/lightningnetwork/lnd/lnwallet/chainfee"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/macaroon.v2"
)

const (
	// 21,000 satoshis
	probeAmount = 21_000

	// The biggest message the client will receive. 200 MiB, the same as `lncli`.
	//
	// Set to be able to get the channel graph when calling the `DescribeGraph` RPC method.
	maxRecvMsgSize = 1024 * 1024 * 200

	// MaxForwardingEvents is the maximum number of forwarding events to get per RPC call.
	MaxForwardingEvents = 50_000
)

// Stream implements a method that receives updates from a stream.
type Stream[T any] interface {
	Recv() (T, error)
}

// Client represents a Lightning Network node client.
type Client interface {
	BatchOpenChannel(ctx context.Context, req *lnrpc.BatchOpenChannelRequest) (string, error)
	CloseChannel(ctx context.Context, req *lnrpc.CloseChannelRequest) (Stream[*lnrpc.CloseStatusUpdate], error)
	ClosedChannels(ctx context.Context) ([]*lnrpc.ChannelCloseSummary, error)
	ConnectPeer(ctx context.Context, publicKey string, addresses []string) error
	DescribeGraph(ctx context.Context) (*lnrpc.ChannelGraph, error)
	EstimateTxFee(ctx context.Context, targetConf int32) (uint64, error)
	EstimateRouteFee(ctx context.Context, publicKey string) (*routerrpc.RouteFeeResponse, error)
	GetChanInfo(ctx context.Context, channelID uint64) (*lnrpc.ChannelEdge, error)
	GetInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error)
	ListChannels(ctx context.Context) ([]*lnrpc.Channel, error)
	ListForwards(ctx context.Context, channelID uint64, startTime, endTime uint64, indexOffset uint32) (*lnrpc.ForwardingHistoryResponse, error)
	ListPeers(ctx context.Context) ([]*lnrpc.Peer, error)
	QueryRoute(ctx context.Context, publicKey string) (*lnrpc.QueryRoutesResponse, error)
	UpdateChannelPolicy(ctx context.Context, channelPoint string, baseFeeMsat, feeRatePPM, maxHTLCMsat, timeLockDelta uint64) error
	WalletBalance(ctx context.Context, minConf int32) (*lnrpc.WalletBalanceResponse, error)
}

type client struct {
	ln     lnrpc.LightningClient
	router routerrpc.RouterClient
	wallet walletrpc.WalletKitClient
}

// NewClient returns a new client that communicates with a Lightning node.
func NewClient(config config.Lightning) (Client, error) {
	logger := logger.New("LND")

	opts, err := loadGRPCOpts(config, logger)
	if err != nil {
		return nil, errors.Wrap(err, "loading gRPC options")
	}

	logger.Infof("Opening gRPC connection to %q", config.RPC.Address)

	conn, err := grpc.NewClient(config.RPC.Address, opts...)
	if err != nil {
		return nil, err
	}

	if err := waitForLND(conn, logger); err != nil {
		return nil, err
	}

	autopilot := autopilotrpc.NewAutopilotClient(conn)
	_, err = autopilot.ModifyStatus(context.Background(), &autopilotrpc.ModifyStatusRequest{Enable: false})
	if err != nil {
		return nil, errors.Wrap(err, "disabling LND's autopilot")
	}

	return &client{
		ln:     lnrpc.NewLightningClient(conn),
		router: routerrpc.NewRouterClient(conn),
		wallet: walletrpc.NewWalletKitClient(conn),
	}, nil
}

func loadGRPCOpts(config config.Lightning, logger logger.Logger) ([]grpc.DialOption, error) {
	logger.Infof("Using TLS certificate %q", config.RPC.TLSCertPath)
	tlsCred, err := credentials.NewClientTLSFromFile(config.RPC.TLSCertPath, "")
	if err != nil {
		return nil, errors.Wrap(err, "unable to read TLS certificate")
	}

	logger.Infof("Using macaroon %q", config.RPC.MacaroonPath)
	macBytes, err := os.ReadFile(config.RPC.MacaroonPath)
	if err != nil {
		return nil, errors.Wrap(err, "reading macaroon file")
	}

	mac := &macaroon.Macaroon{}
	if err := mac.UnmarshalBinary(macBytes); err != nil {
		return nil, errors.Wrap(err, "unmarshaling macaroon")
	}

	macCred, err := macaroons.NewMacaroonCredential(mac)
	if err != nil {
		return nil, errors.Wrap(err, "creating macaroon credential")
	}

	connectionParams := grpc.ConnectParams{
		Backoff:           backoff.DefaultConfig,
		MinConnectTimeout: 5 * time.Second,
	}
	connectionParams.Backoff.MaxDelay = config.RPC.Timeout

	keepAliveParams := keepalive.ClientParameters{
		Time:    10 * time.Second,
		Timeout: config.RPC.Timeout,
	}

	return []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvMsgSize)),
		grpc.WithTransportCredentials(tlsCred),
		grpc.WithPerRPCCredentials(macCred),
		grpc.WithConnectParams(connectionParams),
		grpc.WithKeepaliveParams(keepAliveParams),
	}, nil
}

// waitForLND blocks the execution until LND is fully ready to accept calls.
func waitForLND(conn *grpc.ClientConn, logger logger.Logger) error {
	stateClient := lnrpc.NewStateClient(conn)
	stream, err := stateClient.SubscribeState(context.Background(), &lnrpc.SubscribeStateRequest{})
	if err != nil {
		return errors.Wrap(err, "subscribing to state")
	}

	logger.Info("Waiting for LND to be ready to accept connections")

	for {
		wallet, err := stream.Recv()
		if err != nil {
			return err
		}

		if wallet.State == lnrpc.WalletState_SERVER_ACTIVE {
			logger.Info("The LND server is now active. Initializing connections")
			return nil
		}

		logger.Infof("Wallet state changed to %q", wallet.State)
	}
}

// BatchOpenChannel opens multiple channels in a single on-chain transaction. It returns the final
// transaction ID.
func (c *client) BatchOpenChannel(ctx context.Context, req *lnrpc.BatchOpenChannelRequest) (string, error) {
	resp, err := c.ln.BatchOpenChannel(ctx, req)
	if err != nil {
		return "", err
	}

	txID, err := chainhash.NewHash(resp.PendingChannels[0].Txid)
	if err != nil {
		return "", err
	}

	return txID.String(), nil
}

// CloseChannel closes the specified channel.
func (c *client) CloseChannel(ctx context.Context, req *lnrpc.CloseChannelRequest) (Stream[*lnrpc.CloseStatusUpdate], error) {
	return c.ln.CloseChannel(ctx, req)
}

// ClosedChannels returns a description of all the closed channels that this node was a participant in.
func (c *client) ClosedChannels(ctx context.Context) ([]*lnrpc.ChannelCloseSummary, error) {
	resp, err := c.ln.ClosedChannels(ctx, &lnrpc.ClosedChannelsRequest{
		Cooperative: true,
		LocalForce:  true,
		RemoteForce: true,
		// Breach indicates that the remote peer attempted to broadcast a prior _revoked_ channel state.
		Breach: true,
		// FundingCanceled indicates that there was a failure during the opening workflow or timeout waiting
		// for the funding transaction. The channel was never fully opened.
		//
		// Used to avoid selecting nodes that we have tried opening a channel to recently and failed.
		FundingCanceled: true,
		// Abandoned indicates that the channel state was removed without any further actions.
		Abandoned: false,
	})
	if err != nil {
		return nil, err
	}

	return resp.Channels, nil
}

// ConnectPeer attempts to establish a connection to a remote peer.
func (c *client) ConnectPeer(ctx context.Context, publicKey string, addresses []string) error {
	var connectErr error
	for _, addr := range addresses {
		timeout := uint64(15)
		host, _, _ := strings.Cut(addr, ":")
		if strings.HasSuffix(host, ".onion") {
			timeout = 30
		}

		req := &lnrpc.ConnectPeerRequest{
			Addr: &lnrpc.LightningAddress{
				Pubkey: publicKey,
				Host:   addr,
			},
			Timeout: timeout,
		}
		_, err := c.ln.ConnectPeer(ctx, req)
		if err == nil {
			return nil
		}

		connectErr = err
	}

	return connectErr
}

// DescribeGraph a description of the latest graph state from the point of view of the node.
func (c *client) DescribeGraph(ctx context.Context) (*lnrpc.ChannelGraph, error) {
	return c.ln.DescribeGraph(ctx, &lnrpc.ChannelGraphRequest{IncludeUnannounced: true})
}

// EstimateFee returns the estimated sat/vB cost of mining a transaction.
func (c *client) EstimateTxFee(ctx context.Context, targetConf int32) (uint64, error) {
	fee, err := c.wallet.EstimateFee(ctx, &walletrpc.EstimateFeeRequest{ConfTarget: targetConf})
	if err != nil {
		return 0, err
	}

	// Convert sat/kw to sat/vb
	rateKW := chainfee.SatPerKWeight(fee.SatPerKw)
	return uint64(rateKW.FeePerVByte()), nil
}

// EstimateRouteFee performs an in memory graph estimation and returns how much it may cost to send an HTLC
// to the target end destination.
func (c *client) EstimateRouteFee(ctx context.Context, publicKey string) (*routerrpc.RouteFeeResponse, error) {
	pubKey, err := hex.DecodeString(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "decoding node public key")
	}

	req := &routerrpc.RouteFeeRequest{
		Dest:    pubKey,
		AmtSat:  probeAmount,
		Timeout: 30,
	}
	resp, err := c.router.EstimateRouteFee(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetChanInfo returns the latest authenticated network announcement for the given channel.
func (c *client) GetChanInfo(ctx context.Context, channelID uint64) (*lnrpc.ChannelEdge, error) {
	return c.ln.GetChanInfo(ctx, &lnrpc.ChanInfoRequest{ChanId: channelID})
}

// GetInfo returns general information concerning the lightning node.
func (c *client) GetInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error) {
	return c.ln.GetInfo(ctx, &lnrpc.GetInfoRequest{})
}

// ListChannels returns a description of all the open channels that this node is a participant in.
func (c *client) ListChannels(ctx context.Context) ([]*lnrpc.Channel, error) {
	resp, err := c.ln.ListChannels(ctx, &lnrpc.ListChannelsRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Channels, nil
}

// ListForwards returns list of successful HTLC forwarding events.
func (c *client) ListForwards(
	ctx context.Context,
	channelID uint64,
	startTime,
	endTime uint64,
	indexOffset uint32,
) (*lnrpc.ForwardingHistoryResponse, error) {
	channelIDs := []uint64{channelID}
	if channelID == 0 {
		channelIDs = nil
	}

	return c.ln.ForwardingHistory(ctx, &lnrpc.ForwardingHistoryRequest{
		StartTime:       startTime,
		EndTime:         endTime,
		IndexOffset:     indexOffset,
		NumMaxEvents:    MaxForwardingEvents,
		PeerAliasLookup: false,
		IncomingChanIds: channelIDs,
		OutgoingChanIds: channelIDs,
	})
}

// ListPeers returns a verbose listing of all currently active peers.
func (c *client) ListPeers(ctx context.Context) ([]*lnrpc.Peer, error) {
	resp, err := c.ln.ListPeers(ctx, &lnrpc.ListPeersRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Peers, nil
}

// QueryRoute attempts to query the daemon's Channel Router for a possible route to a target destination
// capable of carrying a specific amount of satoshis.
func (c *client) QueryRoute(ctx context.Context, publicKey string) (*lnrpc.QueryRoutesResponse, error) {
	return c.ln.QueryRoutes(ctx, &lnrpc.QueryRoutesRequest{
		PubKey:            publicKey,
		Amt:               probeAmount,
		UseMissionControl: true,
	})
}

// UpdateChannelPolicy updates the fee schedule and channel policies for a particular channel.
func (c *client) UpdateChannelPolicy(
	ctx context.Context, channelPoint string, baseFeeMsat, feeRatePPM, maxHTLCMsat, timeLockDelta uint64,
) error {
	chanPoint, err := ParseChannelPoint(channelPoint)
	if err != nil {
		return err
	}

	req := &lnrpc.PolicyUpdateRequest{
		Scope: &lnrpc.PolicyUpdateRequest_ChanPoint{
			ChanPoint: chanPoint,
		},
		BaseFeeMsat:   int64(baseFeeMsat),
		FeeRatePpm:    uint32(feeRatePPM),
		MaxHtlcMsat:   maxHTLCMsat,
		TimeLockDelta: uint32(timeLockDelta),
	}
	resp, err := c.ln.UpdateChannelPolicy(ctx, req)
	if err != nil {
		return errors.Wrap(err, "updating channel policy")
	}

	if len(resp.FailedUpdates) > 0 {
		return errors.New(resp.FailedUpdates[0].UpdateError)
	}

	return nil
}

// WalletBalance returns confirmed/unconfirmed and the total number of UTXOs under control of the
// wallet.
func (c *client) WalletBalance(ctx context.Context, minConf int32) (*lnrpc.WalletBalanceResponse, error) {
	return c.ln.WalletBalance(ctx, &lnrpc.WalletBalanceRequest{MinConfs: minConf})
}

// ParseChannelPoint parses a channel point string into the object used by LND.
func ParseChannelPoint(channelPoint string) (*lnrpc.ChannelPoint, error) {
	txID, outpoint, ok := strings.Cut(channelPoint, ":")
	if !ok {
		return nil, errors.New("invalid format")
	}

	if txID == "" {
		return nil, errors.New("invalid transaction ID")
	}

	if outpoint == "" {
		return nil, errors.New("invalid outpoint index")
	}

	outputIndex, err := strconv.ParseUint(outpoint, 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing outpoint index")
	}

	chanPoint := &lnrpc.ChannelPoint{
		FundingTxid: &lnrpc.ChannelPoint_FundingTxidStr{
			FundingTxidStr: txID,
		},
		OutputIndex: uint32(outputIndex),
	}

	return chanPoint, nil
}
