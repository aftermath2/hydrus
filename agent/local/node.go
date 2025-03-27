package local

import (
	"context"
	"encoding/json"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/pkg/errors"
)

// Node contains general information concerning the lightning node.
type Node struct {
	ChannelPeers       map[string]struct{}          `json:"channel_peers,omitempty"`
	SyncPeers          map[string]struct{}          `json:"sync_peers,omitempty"`
	PublicKey          string                       `json:"public_key,omitempty"`
	ClosedChannels     []*lnrpc.ChannelCloseSummary `json:"closed_channels,omitempty"`
	AllocatedBalance   uint64                       `json:"allocated_balance,omitempty"`
	NumChannels        uint64                       `json:"num_channels,omitempty"`
	MaxOpenChannels    uint64                       `json:"max_open_channels,omitempty"`
	MaxCloseChannels   uint64                       `json:"max_close_channels,omitempty"`
	SatvB              uint64                       `json:"sat_vb,omitempty"`
	CurrentBlockHeight uint32                       `json:"current_block_height,omitempty"`
	Channels           Channels                     `json:"channels,omitzero"`
}

func (n Node) String() string {
	localNode, _ := json.Marshal(n)
	return string(localNode)
}

// GetNode returns information about our own node and its channels.
func GetNode(ctx context.Context, config config.Agent, lnd lightning.Client) (Node, error) {
	info, err := lnd.GetInfo(ctx)
	if err != nil {
		return Node{}, errors.Wrap(err, "getting node info")
	}

	if !info.SyncedToGraph {
		return Node{}, errors.New("node is not synced to graph")
	}

	wallet, err := lnd.WalletBalance(ctx, config.ChannelManager.MinConf)
	if err != nil {
		return Node{}, errors.Wrap(err, "getting wallet balance")
	}

	channels, err := lnd.ListChannels(ctx)
	if err != nil {
		return Node{}, errors.Wrap(err, "listing channels")
	}

	numChannels := uint64(info.NumActiveChannels + info.NumPendingChannels + info.NumInactiveChannels)

	channelPeers := make(map[string]struct{}, len(channels))
	for _, channel := range channels {
		channelPeers[channel.RemotePubkey] = struct{}{}
	}

	peers, err := lnd.ListPeers(ctx)
	if err != nil {
		return Node{}, errors.Wrap(err, "listing peers")
	}

	syncPeers := make(map[string]struct{}, len(peers))
	for _, peer := range peers {
		syncPeers[peer.PubKey] = struct{}{}
	}

	closedChannels, err := lnd.ClosedChannels(ctx)
	if err != nil {
		return Node{}, errors.Wrap(err, "listing closed channels")
	}

	satvB, err := lnd.EstimateTxFee(ctx, config.TargetConf)
	if err != nil {
		return Node{}, errors.Wrap(err, "estimating transaction fee")
	}

	chans, err := getChannels(ctx, lnd, config.HeuristicWeights.Close, channels, peers)
	if err != nil {
		return Node{}, err
	}

	allocatedBalance := uint64(wallet.ConfirmedBalance/100) * config.AllocationPercent
	maxOpenChannels := uint64(0)

	if numChannels < config.MaxChannels {
		maxOpenChannels = config.MaxChannels - numChannels

		// If we don't have enough funds to open all the channels, stick to the amount allowed by the
		// allocated balance
		if (maxOpenChannels * config.MinChannelSize) > allocatedBalance {
			maxOpenChannels = allocatedBalance / config.MinChannelSize
		}
	}

	maxCloseChannels := uint64(0)
	if numChannels > config.MinChannels {
		maxCloseChannels = numChannels - config.MinChannels
	}

	return Node{
		CurrentBlockHeight: info.BlockHeight,
		PublicKey:          info.IdentityPubkey,
		AllocatedBalance:   allocatedBalance,
		NumChannels:        numChannels,
		MaxOpenChannels:    maxOpenChannels,
		MaxCloseChannels:   maxCloseChannels,
		ChannelPeers:       channelPeers,
		SyncPeers:          syncPeers,
		ClosedChannels:     closedChannels,
		SatvB:              satvB,
		Channels:           chans,
	}, nil
}
