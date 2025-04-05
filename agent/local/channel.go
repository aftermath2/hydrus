package local

import (
	"context"
	"time"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/graph"
	"github.com/aftermath2/hydrus/lightning"

	"github.com/lightningnetwork/lnd/lnrpc"
)

const (
	oneDay   = time.Hour * 24
	oneWeek  = oneDay * 7
	oneMonth = oneDay * 30
)

// Channels contains a list of public channels and their heuristics.
type Channels struct {
	List       []Channel
	Heuristics Heuristics
}

// Channel represents a public channel the node currently has.
type Channel struct {
	ID              uint64 `json:"id,omitempty"`
	Point           string `json:"point,omitempty"`
	Active          bool   `json:"active,omitempty"`
	BlockHeight     uint32 `json:"block_height,omitempty"`
	RemotePublicKey string `json:"remote_public_key,omitempty"`
	Capacity        uint64 `json:"capacity,omitempty"`
	NumForwards     uint64 `json:"num_forwards,omitempty"`
	ForwardsAmount  uint64 `json:"forwards_amount,omitempty"`
	LocalBalance    uint64 `json:"local_balance,omitempty"`
	Fees            uint64 `json:"fees,omitempty"`
	PingTime        int64  `json:"ping_time,omitempty"`
	FlapCount       int32  `json:"flap_count,omitempty"`
}

// getChannels returns the node's list of public channels along with their heuristics.
func getChannels(
	ctx context.Context,
	lnd lightning.Client,
	closeWeights config.CloseWeights,
	channels []*lnrpc.Channel,
	peers []*lnrpc.Peer,
) (Channels, error) {
	oneMonthAgo := uint64(time.Now().Add(-oneMonth).Unix())
	forwards, err := ListForwards(ctx, lnd, oneMonthAgo, 0)
	if err != nil {
		return Channels{}, err
	}

	heuristics := NewHeuristics(closeWeights)
	chans := make([]Channel, 0, len(channels))
	for _, channel := range channels {
		if channel.Private {
			// Do not close private channels
			continue
		}

		numForwards, forwardsAmount, fees := getForwardsInfo(channel, forwards)
		pingTime, flapCount := getPeerInfo(channel, peers)

		channel := Channel{
			ID:              channel.ChanId,
			BlockHeight:     graph.GetChannelBlockHeight(channel.ChanId),
			Point:           channel.ChannelPoint,
			Active:          channel.Active,
			Capacity:        uint64(channel.Capacity),
			NumForwards:     numForwards,
			ForwardsAmount:  forwardsAmount,
			LocalBalance:    uint64(channel.LocalBalance),
			Fees:            fees,
			RemotePublicKey: channel.RemotePubkey,
			PingTime:        pingTime,
			FlapCount:       flapCount,
		}

		heuristics.Update(channel)
		chans = append(chans, channel)
	}

	return Channels{List: chans, Heuristics: *heuristics}, nil
}

// ListForwards gets all channels forwards by paginating over LND's ListForwards RPC.
func ListForwards(
	ctx context.Context,
	lnd lightning.Client,
	startTime uint64,
	offset uint32,
) ([]*lnrpc.ForwardingEvent, error) {
	events := make([]*lnrpc.ForwardingEvent, 0)
	now := uint64(time.Now().Unix())

	for {
		forwards, err := lnd.ListForwards(ctx, startTime, now, offset)
		if err != nil {
			return nil, err
		}

		events = append(events, forwards.ForwardingEvents...)

		// 50k is the maximum number of events we are getting per request
		if len(forwards.ForwardingEvents) != lightning.MaxForwardingEvents {
			break
		}

		offset = forwards.LastOffsetIndex
	}

	return events, nil
}

func getForwardsInfo(channel *lnrpc.Channel, forwards []*lnrpc.ForwardingEvent) (uint64, uint64, uint64) {
	var numForwards, forwardsAmount, fees uint64
	for _, forward := range forwards {
		if forward.ChanIdIn == channel.ChanId {
			numForwards++
			forwardsAmount += forward.AmtInMsat
			// Even though we collect fees in the other part of the circuit, we are counting fees for this
			// channel as well for opening it
			fees += forward.FeeMsat
		}

		if forward.ChanIdOut == channel.ChanId {
			numForwards++
			forwardsAmount += forward.AmtOutMsat
			fees += forward.FeeMsat
		}
	}

	return numForwards, forwardsAmount, fees
}

func getPeerInfo(channel *lnrpc.Channel, peers []*lnrpc.Peer) (int64, int32) {
	var pingTime int64
	var flapCount int32
	for _, peer := range peers {
		if peer.PubKey == channel.RemotePubkey {
			// Ping time is -1 when we just connected to a peer, set a default value to avoid
			// comparing other values against a negative one
			if peer.PingTime == -1 {
				pingTime = 1500
			} else {
				pingTime = peer.PingTime
			}
			flapCount = peer.FlapCount
			break
		}
	}

	return pingTime, flapCount
}
