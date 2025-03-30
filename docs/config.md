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
	uri:/lnrpc.Lightning/GetInfo \
	uri:/lnrpc.Lightning/ListChannels \
	uri:/lnrpc.Lightning/ListForwards \
	uri:/lnrpc.Lightning/ListPeers \
	uri:/lnrpc.Lightning/QueryRoute \
	uri:/lnrpc.Lightning/WalletBalance
```

> [!Note]
> Executing Hydrus with the CLI flag `--nodes_scores` only requires `uri:/lnrpc.Lightning/DescribeGraph`.

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

| Name | Description |
|------|-------------|
| `agent.dry_run` | Enable dry-run mode to run without making actual changes |
| `agent.blocklist` | A list of public keys to discard when opening channels |
| `agent.keeplist` | A list of public keys to keep when closing channels |
| `agent.allocation_percent` | Wallet balance percentage allocation |
| `agent.allow_force_closes` | Enable channels force-closing |
| `agent.target_conf` | Target confirmation blocks for channel operations |
| `agent.min_batch_size` | Minimum batch size. Used to open at least n channels per transaction |
| `agent.min_channels` | Minimum number of channels required |
| `agent.max_channels` | Maximum number of channels allowed |
| `agent.min_channel_size` | Minimum channel funding amount |
| `agent.max_channel_size` | Maximum channel funding amount |

#### Channel manager

| Name | Description |
|------|-------------|
| `agent.channel_manager.max_sat_vb` | Maximum number of satoshis per virtual byte to create a transaction |
| `agent.channel_manager.min_conf` | Minimum confirmations required to spend a UTXO |
| `agent.channel_manager.base_fee_msat` | New channels initial base fee in milli-satoshis |
| `agent.channel_manager.fee_rate_ppm` | New channel initial fee rate in parts per million (ppm) |

#### Heuristics

##### Open

| Name | Description |
|------|-------------|
| `agent.heuristic_weights.open.capacity` | Nodes capacity weight |
| `agent.heuristic_weights.open.features` | Nodes features weight |
| `agent.heuristic_weights.open.hybrid` | Weight for nodes addresses types |
| `agent.heuristic_weights.open.degree_centrality` | Weight for the degree centrality of the node in the network |
| `agent.heuristic_weights.open.closeness_centrality` | Weight for the closeness centrality of the node |
| `agent.heuristic_weights.open.betweenness_centrality` | Weight for the betweenness centrality of the node |
| `agent.heuristic_weights.open.eigenvector_centrality` | Weight for the eigenvector centrality of the node |
| `agent.heuristic_weights.open.channels.base_fee` | Channels base fees weight |
| `agent.heuristic_weights.open.channels.fee_rate` | Channels fee rate weight |
| `agent.heuristic_weights.open.channels.inbound_base_fee` | Inbound base fee weight |
| `agent.heuristic_weights.open.channels.inbound_fee_rate` | Inbound fee rate weight |
| `agent.heuristic_weights.open.channels.min_htlc` | Minimum HTLC value weight |
| `agent.heuristic_weights.open.channels.max_htlc` | Maximum HTLC value allowed weight |
| `agent.heuristic_weights.open.channels.block_height` | Channels block height (age) weight |

##### Close

| Name | Description |
|------|-------------|
| `agent.heuristic_weights.close.capacity` | Channel capacity |
| `agent.heuristic_weights.close.active` | Channel status |
| `agent.heuristic_weights.close.num_forwards` | Number of forwards weight for channel closures |
| `agent.heuristic_weights.close.forwards_amount` | Total amount forwarded weight for channel closures |
| `agent.heuristic_weights.close.fees` | Fees collected weight for channel closures |
| `agent.heuristic_weights.close.age` | Age of the channel when considering closure |
| `agent.heuristic_weights.close.ping_time` | Ping time to the peer node |
| `agent.heuristic_weights.close.flap_count` | The number of times we have recorded the peer going offline or coming online |
