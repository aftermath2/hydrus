package channel

import (
	"context"
	"encoding/hex"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// OpenRequest contains the information necessary to open a set of channels.
type OpenRequest struct {
	// map[public_key]funding_amount
	Nodes map[string]uint64
	SatvB uint64
}

// CloseRequest contains the information necessary to close a set of channels.
type CloseRequest struct {
	// map[channel_point]force_close
	Channels map[string]bool
	SatvB    uint64
}

// Manager handles the opening, closing an re-sizing of channels.
type Manager interface {
	Open(ctx context.Context, req OpenRequest) error
	Close(ctx context.Context, req CloseRequest) error
	// Splice(channelID uint64, amount int64) error
	UpdatePolicy(ctx context.Context, channelPoint string, feeRatePPM, maxHTLC uint64) error
}

type manager struct {
	lnd           lightning.Client
	logger        logger.Logger
	subscriptions map[string]struct{}
	config        config.ChannelManager
}

// NewManager returns a channel manager that opens, closes and re-sizes channels.
func NewManager(config config.ChannelManager, lnd lightning.Client) Manager {
	return &manager{
		config: config,
		lnd:    lnd,
		logger: logger.New("CHM"),
	}
}

func (m *manager) Open(ctx context.Context, req OpenRequest) error {
	batch := make([]*lnrpc.BatchOpenChannel, 0, len(req.Nodes))

	for publicKey, amount := range req.Nodes {
		pubKey, err := hex.DecodeString(publicKey)
		if err != nil {
			return errors.Wrap(err, "decoding public key")
		}

		batch = append(batch, &lnrpc.BatchOpenChannel{
			NodePubkey:         pubKey,
			LocalFundingAmount: int64(amount),
			BaseFee:            m.config.BaseFeeMsat,
			UseBaseFee:         true,
			FeeRate:            m.config.FeeRatePPM,
			UseFeeRate:         true,
		})
	}

	m.logger.Tracef("Batch open request channels: %#v", batch)

	txID, err := m.lnd.BatchOpenChannel(ctx, &lnrpc.BatchOpenChannelRequest{
		Channels:              batch,
		MinConfs:              m.config.MinConf,
		SatPerVbyte:           int64(req.SatvB),
		SpendUnconfirmed:      false,
		Label:                 "Hydrus",
		CoinSelectionStrategy: lnrpc.CoinSelectionStrategy_STRATEGY_USE_GLOBAL_CONFIG,
	})
	if err != nil {
		return errors.Wrap(err, "batch opening channels")
	}

	m.logger.Infof("Opening channels in transaction %q", txID)
	return nil
}

func (m *manager) Close(ctx context.Context, req CloseRequest) error {
	g, ctx := errgroup.WithContext(ctx)

	for channelPoint, force := range req.Channels {
		g.Go(func() error {
			return m.close(ctx, req.SatvB, channelPoint, force)
		})
	}

	return g.Wait()
}

func (m *manager) close(ctx context.Context, satvB uint64, channelPoint string, force bool) error {
	chanPoint, err := lightning.ParseChannelPoint(channelPoint)
	if err != nil {
		return errors.Wrap(err, "parsing channel point")
	}

	req := &lnrpc.CloseChannelRequest{
		ChannelPoint:   chanPoint,
		SatPerVbyte:    satvB,
		MaxFeePerVbyte: m.config.MaxSatvB,
		Force:          force,
	}

	stream, err := m.lnd.CloseChannel(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "closing channel %q", channelPoint)
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			return errors.Wrap(err, "receiving channel close update")
		}

		pendingUpdate := update.GetClosePending()
		if pendingUpdate != nil {
			txID, err := chainhash.NewHash(pendingUpdate.Txid)
			if err != nil {
				return errors.Wrap(err, "parsing transaction ID")
			}

			m.logger.Infof("Closing channel on outpoint %q in transaction %s",
				channelPoint, txID.String(),
			)
			return nil
		}
	}
}

func (m *manager) UpdatePolicy(ctx context.Context, channelPoint string, feeRatePPM, maxHTLC uint64) error {
	return m.lnd.UpdateChannelPolicy(ctx, channelPoint, feeRatePPM, maxHTLC)
}
