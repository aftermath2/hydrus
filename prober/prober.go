package prober

import (
	"context"
	"time"

	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/lightningnetwork/lnd/lnrpc"
)

// TODO: use probing to determine where liquidity flows to

// Result represents the information obtained by the probe.
type Result struct {
	Success bool
	// Time to reach the node in milliseconds
	PingTime           int64
	ShortestRoute      int64
	CheapestRoute      int64
	SuccessProbability float64
}

// Prober is in charge of executing probes to the target nodes.
type Prober interface {
	Probe(ctx context.Context, publicKey string) (Result, error)
}

type prober struct {
	lnd    lightning.Client
	logger logger.Logger
}

// New returns a new prober.
func New(lnd lightning.Client) Prober {
	return &prober{
		logger: logger.New("PRB"),
		lnd:    lnd,
	}
}

func (p *prober) Probe(ctx context.Context, publicKey string) (Result, error) {
	route, err := p.lnd.QueryRoute(ctx, publicKey)
	if err != nil {
		return Result{}, err
	}

	shortestRoute := getShortestRoute(route.Routes)

	startTime := time.Now()
	probe, err := p.lnd.EstimateRouteFee(ctx, publicKey)
	if err != nil {
		return Result{}, err
	}
	endTime := time.Since(startTime)

	return Result{
		Success:            probe.FailureReason == lnrpc.PaymentFailureReason_FAILURE_REASON_NONE,
		ShortestRoute:      int64(len(shortestRoute.Hops)),
		CheapestRoute:      probe.RoutingFeeMsat,
		PingTime:           endTime.Milliseconds(),
		SuccessProbability: route.SuccessProb,
	}, nil
}

func getShortestRoute(routes []*lnrpc.Route) *lnrpc.Route {
	if len(routes) == 0 {
		return nil
	}

	shortestRoute := routes[0]

	for _, route := range routes[1:] {
		if len(route.Hops) < len(shortestRoute.Hops) {
			shortestRoute = route
		}
	}

	return shortestRoute
}
