package scores

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/cmd"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"
	"github.com/spf13/cobra"

	"github.com/pkg/errors"
)

type candidateChannel struct {
	ID           uint64  `json:"id"`
	ChannelPoint string  `json:"channel_point"`
	Score        float64 `json:"score"`
}

// NewChannelsCmd returns a new scores channels command.
func NewChannelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "channels",
		Short: "Show local channels scores",
		RunE: cmd.Run(func(ctx context.Context, config *config.Config, lnd lightning.Client, logger logger.Logger) error {
			localNode, err := local.GetNode(ctx, config.Agent, lnd)
			if err != nil {
				return errors.Wrap(err, "getting local node")
			}

			if len(localNode.Channels.List) == 0 {
				logger.Info("The node has no channels")
				return nil
			}

			heu, err := json.Marshal(localNode.Channels.Heuristics)
			if err != nil {
				return errors.Wrap(err, "parsing channels heuristics")
			}
			logger.Infof("Local node channels heuristics: %s", heu)

			candidates := make([]candidateChannel, 0, len(localNode.Channels.List))
			for _, channel := range localNode.Channels.List {
				candidates = append(candidates, candidateChannel{
					ID:           channel.ID,
					ChannelPoint: channel.Point,
					Score:        localNode.Channels.Heuristics.GetScore(channel),
				})
			}

			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].Score < candidates[j].Score
			})

			encCandidates, err := json.Marshal(candidates)
			if err != nil {
				return errors.Wrap(err, "encoding candidates list")
			}

			logger.Infof("Scores: %s", encCandidates)
			return nil
		}),
	}
}
