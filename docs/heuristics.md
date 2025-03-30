## Heuristics

All heuristics work the same way, they collect the highest and lowest value of the entire network, and then uses them to get the score of a specific node.

Each one has a different weight that can be tweaked in the configuration, the score is multiplied by the weight to have more/less influence in the final score.

The formula is the following

```bash
score = (node_value - lowest_value) * (1 / (highest_value - lowest_value))
score = score * weight

# If a lower value is better, the score is inversed
score = (1 - score) * weight
```

A node's final score is composed by the sum of all the heuristics scores.

### Default weights

Default weights are located at [config.go](../config/config.go).

#### Open

| Weight | Value |
| -- | -- |
| capacity | 1 |
| features | 1 |
| hybrid | 0.8 |
| centrality.degree | 0.4 |
| centrality.betweenness | 0.8 |
| centrality.eigenvector | 0.5 |
| centrality.closeness | 0.8 |
| channels.base_fee | 1 |
| channels.fee_rate | 0.7 |
| channels.inbound_base_fee | 0.8 |
| channels.inbound_fee_rate | 0.7 |
| channels.min_htlc | 1 |
| channels.max_htlc | 0.6 |
| channels.block_height | 0.8 |

#### Close

| Weight | Value |
| -- | -- |
| capacity | 0.5 |
| active | 1 |
| num_forwards | 0.8 |
| forwards_amount | 1 |
| fees | 1 |
| ping_time | 0.4 |
| block_height | 0.6 |
| flap_count | 0.2 |
