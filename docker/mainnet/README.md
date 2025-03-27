## Hydrus mainnet testing environment

This environment is designed to test hydrus against the mainnet network graph.

It spins up a light neutrino node that adds a peer and loads the graph to then execute hydrus on it.

Make sure the configuration has `dry-run` enabled:

```yml
agent.dry-run: true
```
