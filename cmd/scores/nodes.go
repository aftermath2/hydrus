package scores

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/aftermath2/hydrus/cmd"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/graph"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"
	"github.com/spf13/cobra"

	"github.com/pkg/errors"
)

type candidateNode struct {
	Alias string  `json:"alias"`
	Score float64 `json:"score"`
}

// NewNodesCmd returns a new scores nodes command.
func NewNodesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nodes",
		Short: "Show network graph nodes scores",
		RunE: cmd.Run(func(ctx context.Context, config *config.Config, lnd lightning.Client, logger logger.Logger) error {
			networkGraph, err := graph.New(ctx, config.Agent.HeuristicWeights.Open, lnd)
			if err != nil {
				return errors.Wrap(err, "creating graph")
			}

			heu, err := json.Marshal(networkGraph.Heuristics)
			if err != nil {
				return errors.Wrap(err, "parsing graph heuristics")
			}
			logger.Infof("Network heuristics: %s", heu)

			candidates := make([]candidateNode, 0, len(networkGraph.Nodes))
			for _, node := range networkGraph.Nodes {
				candidates = append(candidates, candidateNode{
					Alias: node.Alias,
					Score: networkGraph.Heuristics.GetScore(node),
				})
			}

			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].Score > candidates[j].Score
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
