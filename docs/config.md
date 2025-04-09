## Configuration

The configuration is quite simple, you will only need the LND node RPC address, a TLS certificate that will secure the communication and a macaroon file to authenticate and execute RPC calls.

To specify the path to the configuration file, use the `--config` CLI flag, if it is not set, Hydrus will look for the configuration file using the `HYDRUS_CONFIG` environment variable, if the variable is not specified, it will search for it at `HOME/hydrus.yml`.

Sample configuration files with just the required fields and with all fields can be found at [sample_config_required.yml](./config/sample_config_required.yml) and [sample_config.yml](./config/sample_config.yml)

### Macaroons

Hydrus needs a macaroon to communicate with the LND instance to manage channels and get information about the node.

To generate a fine-grained macaroon that grants access to the RPC methods used by Hydrus, execute the following command:

```
lncli bakemacaroon --save_to hydrus.macaroon \
	uri:/lnrpc.Lightning/BatchOpenChannel \
	uri:/lnrpc.Lightning/CloseChannel \
	uri:/lnrpc.Lightning/ClosedChannels \
	uri:/lnrpc.Lightning/ConnectPeer \
	uri:/lnrpc.Lightning/DescribeGraph \
	uri:/walletrpc.WalletKit/EstimateTxFee \
	uri:/routerrpc.Router/EstimateRouteFee \
	uri:/lnrpc.Lightning/GetChanInfo \
	uri:/lnrpc.Lightning/GetInfo \
	uri:/lnrpc.Lightning/ListChannels \
	uri:/lnrpc.Lightning/ListForwards \
	uri:/lnrpc.Lightning/ListPeers \
	uri:/lnrpc.Lightning/QueryRoute \
	uri:/lnrpc.Lightning/UpdateChannelPolicy \
	uri:/lnrpc.Lightning/WalletBalance
```

> [!Note]
> Executing Hydrus with the CLI flag `--nodes_scores` only requires `uri:/lnrpc.Lightning/DescribeGraph`.

## Intervals

Hydrus is designed to be executed on a regular basis to keep your node connected to the best peers and close channels that are not performing well. Similarly, routing policies must be updated frequently to manage liquidity efficiently.

This is why the agent has two configurations `intervals.channels` and `intervals.routing_policies` for users to modify and pick the timeframe they are more comfortable with. 

The `agent run` command will execute channels and routing policies adjustments following those intervals forever, blocking the program execution between idle times. 

It is recommended to not run update channels more than once a week, because there isn't much time for the scores to change in a short timeframe, and the agent will end up spending more in fees as the channels opening batches will be smaller. Good timeframes are probably weekly, semi-monthly or monthly.

On the other hand, updating routing policies more than once every hour will cause peers to blacklist the node to avoid spam. Suggested intervals are 3/6/12/24 hours.

## Options

### Lightning

| Name | Type | Description |
|------|------|-------------|
| `lightning.rpc.address` | string | The address where the RPC server is bound to |
| `lightning.rpc.tls_cert_path` | string | Path to the TLS certificate file for RPC communication |
| `lightning.rpc.macaroon_path` | string | Path to the macaroon authentication file for RPC communication |
| `lightning.rpc.timeout` | time.Duration | Timeout duration for RPC requests |

### Logging

| Name | Type | Description |
|------|------|-------------|
| `logging.level` | string | Logging level for the application |

### Agent

| Name | Type | Description |
|------|------|-------------|
| `agent.dry_run` | boolean | Enable dry-run mode to run without making actual changes |
| `agent.blocklist` | []string | A list of public keys to discard when opening channels |
| `agent.keeplist` | []string | A list of public keys to keep when closing channels |
| `agent.allocation_percent` | int | Wallet balance percentage allocation |
| `agent.allow_force_closes` | boolean | Enable channels force-closing |
| `agent.target_conf` | int | Target confirmation blocks for channel operations |
| `agent.min_batch_size` | int | Minimum batch size. Used to open at least n channels per transaction |
| `agent.min_channels` | int | Minimum number of channels required |
| `agent.max_channels` | int | Maximum number of channels allowed |
| `agent.min_channel_size` | int | Minimum channel funding amount |
| `agent.max_channel_size` | int | Maximum channel funding amount |

#### Channel manager

| Name | Type | Description |
|------|------|-------------|
| `agent.channel_manager.max_sat_vb` | int | Maximum number of satoshis per virtual byte to create a transaction |
| `agent.channel_manager.min_conf` | int | Minimum confirmations required to spend a UTXO |
| `agent.channel_manager.base_fee_msat` | int | New channels initial base fee in milli-satoshis |
| `agent.channel_manager.fee_rate_ppm` | int | New channel initial fee rate in parts per million (ppm) |

#### Heuristics

##### Open

| Name | Type | Description |
|------|------|-------------|
| `agent.heuristic_weights.open.capacity` | float | Nodes capacity weight |
| `agent.heuristic_weights.open.features` | float | Nodes features weight |
| `agent.heuristic_weights.open.hybrid` | float | Weight for nodes addresses types |
| `agent.heuristic_weights.open.centrality.degree` | float | Weight for the degree centrality of the node in the network |
| `agent.heuristic_weights.open.centrality.closeness` | float | Weight for the closeness centrality of the node |
| `agent.heuristic_weights.open.centrality.betweenness` | float | Weight for the betweenness centrality of the node |
| `agent.heuristic_weights.open.centrality.eigenvector` | float | Weight for the eigenvector centrality of the node |
| `agent.heuristic_weights.open.channels.base_fee` | float | Channels base fees weight |
| `agent.heuristic_weights.open.channels.fee_rate` | float | Channels fee rate weight |
| `agent.heuristic_weights.open.channels.inbound_base_fee` | float | Inbound base fee weight |
| `agent.heuristic_weights.open.channels.inbound_fee_rate` | float | Inbound fee rate weight |
| `agent.heuristic_weights.open.channels.min_htlc` | float | Minimum HTLC value weight |
| `agent.heuristic_weights.open.channels.max_htlc` | float | Maximum HTLC value allowed weight |
| `agent.heuristic_weights.open.channels.block_height` | float | Channels block height weight |

##### Close

| Name | Type | Description |
|------|------|-------------|
| `agent.heuristic_weights.close.capacity` | float | Channel capacity |
| `agent.heuristic_weights.close.active` | float | Channel status |
| `agent.heuristic_weights.close.num_forwards` | float | Weight for the number of forwards the channel has routed |
| `agent.heuristic_weights.close.forwards_amount` | float | Total amount forwarded weight |
| `agent.heuristic_weights.close.fees` | float | Fees collected weight |
| `agent.heuristic_weights.close.block_height` | float | Opening transaction block height weight |
| `agent.heuristic_weights.close.ping_time` | float | Ping time to the peer node |
| `agent.heuristic_weights.close.flap_count` | float | The number of times we have recorded the peer going offline or coming online |

##### Intervals

| Name | Type | Description |
|------|------|-------------|
| `agent.intervals.channels` | time | Channels modifications interval |
| `agent.intervals.routing_policies` | time | Routing policies modifications interval |

##### Routing policies

| Name | Type | Description |
|------|------|-------------|
| `agent.routing_policies.forwards.activity_period` | time | Time period for the forwards that are considered to adjust the channels routing policies |

> [!Note]
> Time values support the units "ns", "µs", "ms", "s", "m", "h".
