## Hydrus

Lightning liquidity management agent. Enjoy self-custodial lightning without worrying about channels.

### Getting started

To install and configure Hydrus, check out the [install](docs/install/install.md) and [config](docs/config.md) guides.

> [!Note]
> LND is the only lightning implementation supported for now.

> [!Warning]
> This is beta software, use it at your own risk. If you want to see what Hydrus would actually do without moving any funds, set the `agent.dry_run` field to `true` in the configuration and create a macaroon without permissions to open/close channels.

### How it works

Hydrus opens channels automatically based on the state of the network graph. It also closes and re-sizes (planned for future releases) existing ones based on their characteristics (status, capacity, age) and past performance (number of forwards, fees collected, uptime and more).

It is completely stateless, uses only the information provided by the lightning daemon RPC API.

## Channel opening

Hydrus constructs the network graph in memory and calculate its statistics to obtain the best scores possible for each one of the heuristics. 

It then compares each node in the graph against the best values to calculate its score and ranks them based on that.

Lastly, it iterates through that ranking from highest to lowest scores and try to add the node as a peer, if the connection is successful, it opens a channel to it.

### Batch transactions

Channels are opened using batch transactions, multiple channels are opened in the same on-chain transaction in an atomic way. This means that either all channels are opened at once or they are aborted if any of them fail.

A common reason why channel openings fail is because the remote party has a minimum channel size requirement higher than our funding amount. In these cases, Hydrus will throw an error and discard that node for the next 3 months to avoid failing for the same reason in future executions. To avoid the failure altogether, consider increasing your minimum channel size (`agent.min_channel_size`) or allocating more funds to your node.

To take advantage of batching and to avoid creating a transaction for a few channels, increase the `agent.min_batch_size` value in the configuration.

### Nodes selection algorithm

To select the best candidates to open a channel to, Hydrus takes several factors into consideration:

- **Capacity**: Node total capacity.
- **Centrality**: Network centrality heuristics (see [docs/centrality.md](docs/centrality.md)).
- **Channels**: Node number of active channels and their age, disabled ones are not taken into account.
- **Routing policies**: The node's channels fees and HTLC ranges.
- **Connectivity**: Whether the node is available on both clearnet and tor and the time to reach it.
- **Features**: Supported features.
- **Closed channels**: Nodes we have recently closed a channel with will be discarded.

The algorithm will avoid opening channels to nodes that are already connected to the host or nodes that recently closed channels the host had with them.

#### Weighted heuristics

Each node heuristic has a weight that impacts on the final score result, weights can be modified in the configuration and the default values are carefully chosen to provide the best experience without having to modify them.

> For example, having low fees or supporting a wide range of HTLC values is not necessarily good, as the channel could be depleted. In some cases, fees are increased and HTLC ranges adjusted to improve the node's forwarding reliability.

This way, the algorithm can be tweaked to accommodate to the user's specific needs or to value certain characteristics more than others.

Learn more about heuristics and how scores are calculated at [heuristics.md](docs/heuristics.md).

## Channel closing

To close channels, Hydrus analyzes the heuristics of every local channel and picks those with the worst scores. During this analysis, it takes into account the following factors:

- **Capacity**: Total capacity.
- **Status**: Whether the channel is enabled or disabled.
- **Number of forwards**: Number of HTLCs routed through the channel.
- **Forwards amount**: Bitcoin amount routed through the channel.
- **Fees**: Fees collected by the channel.
- **Ping time**: Time to reach the peer node.
- **Block height**: Opening transaction block height, older channels are preferred.
- **Flap count**: Number of times we lost connection with the peer.

It will only close channels if the number of them is higher than `agent.min_channels` and if their score is lower than 0.3 in a scale of 0 to 1.

## Routing policies

Channels routing policies are adjusted based on the channel state (capacity, local balance) and the amount of satoshis forwarded in the last activity period.

- If the local balance is lower than 1% of the channel capacity, the fee rate is set to 2,100 ppm to discourage routing thorugh the channel.
- If the local balance is higher than 99% of the channel capacity, the fee rate is set to 0 ppm to incentivize routing thorugh the channel.
- If the amount of satoshis forwarded out through the channel is zero, the fee rate is decreased by 10%.
- If the ratio forwards amount out/in is below 0.5, the fee rate is decreased proportionally to half the ratio.
- If the ratio forwards amount out/in is above 0.5, the fee rate is increased proportionally to half the ratio.

> For example, if the forwards ratio is 0.8, meaning that we are sending more payments than receiving, the fee rate will be increased by 30% (0.8 - 0.5). Similarly, if the forwards ratio is 0.4, the fee rate will be decreased by 0.1% (0.5 - 0.4).

Apart from this, Hydrus adjusts the maximum HTLC value to 80% of the local balance. It is not set to the exact local balance to leave a buffer of 20% of the funds, in order to avoid running out of liquidity and failing to route payments before the next update.

## Differences with LND's autopilot

- Autopilot only considers a few centrality heuristics to open channels, Hydrus evaluates many more, as we have seen [above](#nodes-selection-algorithm).
- Autopilot does not batch channel openings, for *n* new channels it uses *n* on-chain transactions instead of only one.
- Autopilot currently does not close channels.
- Autopilot's configuration is limited, it does not support blocking/keeping certain nodes or weighting heuristics, for example.
- The Lightning Labs team has higher priority issues and features to work on, the autopilot module hasn't received enhancements in 3 years. We could expect Hydrus to have far more experimental and novel features.
