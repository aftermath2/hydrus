## Hydrus mainnet testing environment

This environment is designed to test hydrus against the mainnet network graph.

It spins up a lightweight neutrino node that adds a peer and loads the graph to then execute hydrus on it.

Make sure the configuration has `dry-run` enabled:

```yml
agent.dry_run: true
```

### Commands

#### Build the environment

```console
docker compose build
```

#### Start the environment

```console
docker compose up -d
```

#### Stop the environment

```console
docker compose stop
```

#### Restart hydrus

```console
docker restart hydrus
```

#### Show a hydrus logs

```console
docker logs -f hydrus
```
