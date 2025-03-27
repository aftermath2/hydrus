package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"sort"

	"github.com/aftermath2/hydrus/agent"
	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/graph"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/pkg/errors"
)

func main() {
	var (
		channelsScores, nodesScores bool
		configPath                  string
	)
	flag.StringVar(&configPath, "config", "", "Path to the configuration file")
	flag.BoolVar(&channelsScores, "channels_scores", false, "Display node's active channels scores")
	flag.BoolVar(&nodesScores, "nodes_scores", false, "Display network graph nodes scores")
	flag.Parse()

	config, err := config.Load(configPath)
	if err != nil {
		log.Fatal(err)
	}

	lnd, err := lightning.NewClient(config.Lightning)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	if channelsScores {
		if err := printChannelsScores(ctx, config, lnd); err != nil {
			log.Fatal(err)
		}
		return
	}

	if nodesScores {
		if err := printNodesScores(ctx, config, lnd); err != nil {
			log.Fatal(err)
		}
		return
	}

	agent := agent.New(config.Agent, lnd)

	if err := agent.Start(ctx); err != nil {
		log.Fatal(err)
	}
}

func printChannelsScores(ctx context.Context, config *config.Config, lnd lightning.Client) error {
	localNode, err := local.GetNode(ctx, config.Agent, lnd)
	if err != nil {
		return errors.Wrap(err, "getting local node")
	}

	heu, _ := json.Marshal(localNode.Channels.Heuristics)
	logger := logger.New("HYD")
	logger.Infof("Local node channels heuristics: %s", heu)

	type candidate struct {
		ID           uint64  `json:"id"`
		ChannelPoint string  `json:"channel_point"`
		Score        float64 `json:"score"`
	}

	candidates := make([]candidate, 0, len(localNode.Channels.List))
	for _, channel := range localNode.Channels.List {
		candidates = append(candidates, candidate{
			ID:           channel.ID,
			ChannelPoint: channel.Point,
			Score:        localNode.Channels.Heuristics.GetScore(channel),
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score < candidates[j].Score
	})

	candidatesB, _ := json.Marshal(candidates)
	logger.Infof("Scores: %s", candidatesB)

	return nil
}

func printNodesScores(ctx context.Context, config *config.Config, lnd lightning.Client) error {
	networkGraph, err := graph.New(ctx, config.Agent.HeuristicWeights.Open, lnd)
	if err != nil {
		return errors.Wrap(err, "creating graph")
	}

	heu, _ := json.Marshal(networkGraph.Heuristics)
	logger := logger.New("HYD")
	logger.Infof("Network heuristics: %s", heu)

	type candidate struct {
		Alias string  `json:"alias"`
		Score float64 `json:"score"`
	}

	candidates := make([]candidate, 0, len(networkGraph.Nodes))
	for _, node := range networkGraph.Nodes {
		candidates = append(candidates, candidate{
			Alias: node.Alias,
			Score: networkGraph.Heuristics.GetScore(node),
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	candidatesB, _ := json.Marshal(candidates)
	logger.Infof("Scores: %s", candidatesB)

	return nil
}
